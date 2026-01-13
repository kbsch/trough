package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/kbsch/trough/internal/domain"
	"github.com/kbsch/trough/internal/repository"
	"github.com/kbsch/trough/internal/scraper/engine"
)

// ScrapeJobArgs are the arguments for a scrape job
type ScrapeJobArgs struct {
	SourceSlug  string `json:"source_slug"`
	MaxListings int    `json:"max_listings"`
	FullScrape  bool   `json:"full_scrape"`
}

func (ScrapeJobArgs) Kind() string { return "scrape" }

// ScrapeJobWorker handles scraping jobs
type ScrapeJobWorker struct {
	river.WorkerDefaults[ScrapeJobArgs]
	engine      *engine.Engine
	sourceRepo  *repository.SourceRepository
	listingRepo *repository.ListingRepository
}

func NewScrapeJobWorker(eng *engine.Engine, sourceRepo *repository.SourceRepository, listingRepo *repository.ListingRepository) *ScrapeJobWorker {
	return &ScrapeJobWorker{
		engine:      eng,
		sourceRepo:  sourceRepo,
		listingRepo: listingRepo,
	}
}

func (w *ScrapeJobWorker) Work(ctx context.Context, job *river.Job[ScrapeJobArgs]) error {
	args := job.Args
	log.Printf("Starting scrape job for source: %s", args.SourceSlug)

	source, err := w.sourceRepo.GetBySlug(ctx, args.SourceSlug)
	if err != nil {
		return fmt.Errorf("source not found: %s", args.SourceSlug)
	}

	// Create a scrape job record
	scrapeJob := &domain.ScrapeJob{
		ID:        uuid.New(),
		SourceID:  source.ID,
		Status:    domain.ScrapeJobStatusRunning,
		CreatedAt: time.Now(),
	}
	now := time.Now()
	scrapeJob.StartedAt = &now

	if err := w.sourceRepo.CreateScrapeJob(ctx, scrapeJob); err != nil {
		log.Printf("Warning: failed to create scrape job record: %v", err)
	}

	// Run the scraper
	err = w.engine.RunSource(ctx, args.SourceSlug, args.MaxListings)

	// Update job status
	completedAt := time.Now()
	scrapeJob.CompletedAt = &completedAt
	if err != nil {
		scrapeJob.Status = domain.ScrapeJobStatusFailed
		scrapeJob.ErrorMessage = err.Error()
	} else {
		scrapeJob.Status = domain.ScrapeJobStatusCompleted
	}

	if updateErr := w.sourceRepo.UpdateScrapeJob(ctx, scrapeJob); updateErr != nil {
		log.Printf("Warning: failed to update scrape job record: %v", updateErr)
	}

	return err
}

// ScrapeAllJobArgs triggers scraping all active sources
type ScrapeAllJobArgs struct{}

func (ScrapeAllJobArgs) Kind() string { return "scrape_all" }

type ScrapeAllJobWorker struct {
	river.WorkerDefaults[ScrapeAllJobArgs]
	engine      *engine.Engine
	sourceRepo  *repository.SourceRepository
	listingRepo *repository.ListingRepository
}

func NewScrapeAllJobWorker(eng *engine.Engine, sourceRepo *repository.SourceRepository, listingRepo *repository.ListingRepository) *ScrapeAllJobWorker {
	return &ScrapeAllJobWorker{
		engine:      eng,
		sourceRepo:  sourceRepo,
		listingRepo: listingRepo,
	}
}

func (w *ScrapeAllJobWorker) Work(ctx context.Context, job *river.Job[ScrapeAllJobArgs]) error {
	log.Println("Starting scrape all job - running all scrapers sequentially")

	// Instead of queuing individual jobs, just run them all directly
	return w.engine.RunAll(ctx)
}
