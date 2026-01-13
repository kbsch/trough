package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Source struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Slug        string          `json:"slug" db:"slug"`
	BaseURL     string          `json:"base_url" db:"base_url"`
	ScraperType string          `json:"scraper_type" db:"scraper_type"` // "colly" or "rod"
	IsActive    bool            `json:"is_active" db:"is_active"`
	Config      json.RawMessage `json:"config" db:"config"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

type ScrapeJob struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	SourceID        uuid.UUID  `json:"source_id" db:"source_id"`
	Status          string     `json:"status" db:"status"` // pending, running, completed, failed
	StartedAt       *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ListingsFound   int        `json:"listings_found" db:"listings_found"`
	ListingsNew     int        `json:"listings_new" db:"listings_new"`
	ListingsUpdated int        `json:"listings_updated" db:"listings_updated"`
	ErrorMessage    string     `json:"error_message,omitempty" db:"error_message"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

const (
	ScrapeJobStatusPending   = "pending"
	ScrapeJobStatusRunning   = "running"
	ScrapeJobStatusCompleted = "completed"
	ScrapeJobStatusFailed    = "failed"
)

const (
	ScraperTypeColly = "colly"
	ScraperTypeRod   = "rod"
)

// ScrapeOptions configures a scraping run
type ScrapeOptions struct {
	FullScrape   bool
	MaxListings  int
	RateLimit    time.Duration
	LastScrapeAt time.Time
}
