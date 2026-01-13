package api

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kbsch/trough/internal/api/handlers"
	mw "github.com/kbsch/trough/internal/api/middleware"
	"github.com/kbsch/trough/internal/repository"
)

type Server struct {
	router      *chi.Mux
	db          *sqlx.DB
	listingRepo *repository.ListingRepository
	sourceRepo  *repository.SourceRepository
}

func NewServer(db *sqlx.DB) *Server {
	s := &Server{
		router:      chi.NewRouter(),
		db:          db,
		listingRepo: repository.NewListingRepository(db),
		sourceRepo:  repository.NewSourceRepository(db),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	r := s.router

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mw.Metrics)           // Prometheus metrics
	r.Use(mw.StructuredLogger)  // JSON structured logging
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health and readiness checks
	r.Get("/health", s.healthCheck)
	r.Get("/ready", s.readinessCheck)

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Get database URL for handlers that need it
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trough:trough@localhost:5432/trough?sslmode=disable"
	}

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		listingHandler := handlers.NewListingHandler(s.listingRepo)
		sourceHandler := handlers.NewSourceHandler(s.sourceRepo, dbURL)

		// Listings
		r.Get("/listings", listingHandler.Search)
		r.Get("/listings/map", listingHandler.MapView)
		r.Get("/listings/{id}", listingHandler.GetByID)
		r.Get("/filters", listingHandler.GetFilters)

		// Sources
		r.Get("/sources", sourceHandler.List)
		r.Post("/refresh", sourceHandler.TriggerRefresh)
		r.Get("/scrape-jobs", sourceHandler.GetScrapeJobs)
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check database connection
	var dbOK bool
	var dbLatency time.Duration
	dbStart := time.Now()
	if err := s.db.PingContext(ctx); err == nil {
		dbOK = true
		dbLatency = time.Since(dbStart)
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !dbOK {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	// Get memory stats
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"checks": map[string]interface{}{
			"database": map[string]interface{}{
				"status":     dbOK,
				"latency_ms": dbLatency.Milliseconds(),
			},
		},
		"system": map[string]interface{}{
			"goroutines":    runtime.NumGoroutine(),
			"memory_alloc":  mem.Alloc,
			"memory_sys":    mem.Sys,
			"gc_cycles":     mem.NumGC,
		},
		"time": time.Now().UTC(),
	})
}

func (s *Server) readinessCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if database is ready for queries
	var listingCount int
	err := s.db.GetContext(ctx, &listingCount, "SELECT COUNT(*) FROM listings LIMIT 1")

	ready := err == nil
	status := "ready"
	statusCode := http.StatusOK

	if !ready {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"ready":  ready,
		"time":   time.Now().UTC(),
	})
}
