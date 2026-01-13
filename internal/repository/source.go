package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/kbsch/trough/internal/domain"
)

type SourceRepository struct {
	db *sqlx.DB
}

func NewSourceRepository(db *sqlx.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

func (r *SourceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	var source domain.Source
	err := r.db.GetContext(ctx, &source, "SELECT * FROM sources WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *SourceRepository) GetBySlug(ctx context.Context, slug string) (*domain.Source, error) {
	var source domain.Source
	err := r.db.GetContext(ctx, &source, "SELECT * FROM sources WHERE slug = $1", slug)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *SourceRepository) ListActive(ctx context.Context) ([]domain.Source, error) {
	var sources []domain.Source
	err := r.db.SelectContext(ctx, &sources, "SELECT * FROM sources WHERE is_active = true ORDER BY name")
	if err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *SourceRepository) Create(ctx context.Context, source *domain.Source) error {
	query := `
		INSERT INTO sources (id, name, slug, base_url, scraper_type, is_active, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		source.ID, source.Name, source.Slug, source.BaseURL,
		source.ScraperType, source.IsActive, source.Config,
		source.CreatedAt, source.UpdatedAt,
	)
	return err
}

func (r *SourceRepository) CreateScrapeJob(ctx context.Context, job *domain.ScrapeJob) error {
	query := `
		INSERT INTO scrape_jobs (id, source_id, status, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query, job.ID, job.SourceID, job.Status, job.CreatedAt)
	return err
}

func (r *SourceRepository) UpdateScrapeJob(ctx context.Context, job *domain.ScrapeJob) error {
	query := `
		UPDATE scrape_jobs SET
			status = $2,
			started_at = $3,
			completed_at = $4,
			listings_found = $5,
			listings_new = $6,
			listings_updated = $7,
			error_message = $8
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Status, job.StartedAt, job.CompletedAt,
		job.ListingsFound, job.ListingsNew, job.ListingsUpdated,
		job.ErrorMessage,
	)
	return err
}

func (r *SourceRepository) GetRecentScrapeJobs(ctx context.Context, limit int) ([]domain.ScrapeJob, error) {
	var jobs []domain.ScrapeJob
	err := r.db.SelectContext(ctx, &jobs, `
		SELECT sj.*, s.name as source_name
		FROM scrape_jobs sj
		JOIN sources s ON s.id = sj.source_id
		ORDER BY sj.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}
