package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/kbsch/trough/internal/api/middleware"
	"github.com/kbsch/trough/internal/repository"
	"github.com/kbsch/trough/internal/scraper/jobs"
)

type SourceHandler struct {
	repo        *repository.SourceRepository
	dbURL       string
	rateLimiter *middleware.RateLimiter
}

func NewSourceHandler(repo *repository.SourceRepository, dbURL string) *SourceHandler {
	return &SourceHandler{
		repo:        repo,
		dbURL:       dbURL,
		rateLimiter: middleware.NewRateLimiter(1, time.Hour), // 1 request per hour
	}
}

func (h *SourceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sources, err := h.repo.ListActive(ctx)
	if err != nil {
		InternalError(w, r, "Failed to fetch sources")
		return
	}

	// Transform to public response (hide internal config)
	type publicSource struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Slug      string    `json:"slug"`
		BaseURL   string    `json:"base_url"`
		IsActive  bool      `json:"is_active"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	result := make([]publicSource, len(sources))
	for i, s := range sources {
		result[i] = publicSource{
			ID:        s.ID.String(),
			Name:      s.Name,
			Slug:      s.Slug,
			BaseURL:   s.BaseURL,
			IsActive:  s.IsActive,
			UpdatedAt: s.UpdatedAt,
		}
	}

	Success(w, map[string]interface{}{
		"sources": result,
	})
}

func (h *SourceHandler) TriggerRefresh(w http.ResponseWriter, r *http.Request) {
	// Rate limit: 1 refresh per hour per IP
	clientIP := r.RemoteAddr
	if !h.rateLimiter.Allow(clientIP) {
		TooManyRequests(w, r, "Refresh is limited to once per hour. Please try again later.")
		return
	}

	ctx := r.Context()

	// Parse request body for optional source filter
	sourceSlug := r.URL.Query().Get("source")

	// Queue the scrape job
	if err := h.queueScrapeJob(ctx, sourceSlug); err != nil {
		InternalError(w, r, "Failed to queue refresh job")
		return
	}

	message := "Refresh job queued for all sources"
	if sourceSlug != "" {
		message = "Refresh job queued for " + sourceSlug
	}

	Accepted(w, map[string]interface{}{
		"message": message,
		"status":  "queued",
	})
}

func (h *SourceHandler) queueScrapeJob(ctx context.Context, sourceSlug string) error {
	pool, err := pgxpool.New(ctx, h.dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return err
	}

	if sourceSlug == "" {
		_, err = client.Insert(ctx, jobs.ScrapeAllJobArgs{}, nil)
	} else {
		_, err = client.Insert(ctx, jobs.ScrapeJobArgs{
			SourceSlug: sourceSlug,
			FullScrape: false, // Incremental for on-demand
		}, nil)
	}

	return err
}

// GetScrapeJobs returns recent scrape job history
func (h *SourceHandler) GetScrapeJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	jobs, err := h.repo.GetRecentScrapeJobs(ctx, 20)
	if err != nil {
		InternalError(w, r, "Failed to fetch scrape jobs")
		return
	}

	Success(w, map[string]interface{}{
		"jobs": jobs,
	})
}
