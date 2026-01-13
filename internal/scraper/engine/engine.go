package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/kbsch/trough/internal/domain"
	"github.com/kbsch/trough/internal/repository"
)

type Engine struct {
	sourceRepo  *repository.SourceRepository
	listingRepo *repository.ListingRepository
	scrapers    map[string]Scraper
}

type Scraper interface {
	Name() string
	Scrape(ctx context.Context, opts domain.ScrapeOptions) (<-chan *domain.Listing, <-chan error)
}

func NewEngine(sourceRepo *repository.SourceRepository, listingRepo *repository.ListingRepository) *Engine {
	e := &Engine{
		sourceRepo:  sourceRepo,
		listingRepo: listingRepo,
		scrapers:    make(map[string]Scraper),
	}

	return e
}

func (e *Engine) RegisterScraper(name string, scraper Scraper) {
	e.scrapers[name] = scraper
}

func (e *Engine) RunAll(ctx context.Context) error {
	sources, err := e.sourceRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sources: %w", err)
	}

	for _, source := range sources {
		if err := e.RunSource(ctx, source.Slug, 0); err != nil {
			log.Printf("Error scraping %s: %v", source.Slug, err)
		}
	}

	return nil
}

func (e *Engine) RunSource(ctx context.Context, slug string, limit int) error {
	source, err := e.sourceRepo.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("source not found: %s", slug)
	}

	scraper, ok := e.scrapers[slug]
	if !ok {
		return fmt.Errorf("no scraper registered for: %s", slug)
	}

	// Create scrape job
	job := &domain.ScrapeJob{
		ID:        uuid.New(),
		SourceID:  source.ID,
		Status:    domain.ScrapeJobStatusRunning,
		CreatedAt: time.Now(),
	}
	now := time.Now()
	job.StartedAt = &now

	if err := e.sourceRepo.CreateScrapeJob(ctx, job); err != nil {
		log.Printf("Warning: failed to create scrape job: %v", err)
	}

	opts := domain.ScrapeOptions{
		FullScrape:  true,
		MaxListings: limit,
		RateLimit:   2 * time.Second,
	}

	listings, errors := scraper.Scrape(ctx, opts)

	var found, created, updated int

	for {
		select {
		case listing, ok := <-listings:
			if !ok {
				// Channel closed, done
				goto done
			}

			found++
			listing.SourceID = source.ID
			listing.LastSeenAt = time.Now()

			if listing.ID == uuid.Nil {
				listing.ID = uuid.New()
				listing.FirstSeenAt = time.Now()
				created++
			} else {
				updated++
			}

			if err := e.listingRepo.Upsert(ctx, listing); err != nil {
				log.Printf("Error upserting listing %s: %v", listing.ExternalID, err)
			}

		case err, ok := <-errors:
			if !ok {
				continue
			}
			log.Printf("Scrape error: %v", err)
		}
	}

done:
	// Update job status
	completedAt := time.Now()
	job.Status = domain.ScrapeJobStatusCompleted
	job.CompletedAt = &completedAt
	job.ListingsFound = found
	job.ListingsNew = created
	job.ListingsUpdated = updated

	if err := e.sourceRepo.UpdateScrapeJob(ctx, job); err != nil {
		log.Printf("Warning: failed to update scrape job: %v", err)
	}

	log.Printf("Scrape completed for %s: found=%d, new=%d, updated=%d",
		slug, found, created, updated)

	return nil
}
