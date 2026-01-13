package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/kbsch/trough/internal/repository"
	"github.com/kbsch/trough/internal/scraper/engine"
	"github.com/kbsch/trough/internal/scraper/jobs"
	"github.com/kbsch/trough/internal/scraper/sources"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trough:trough@localhost:5432/trough?sslmode=disable"
	}

	// Connection for sqlx (repositories)
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connection pool for River (pgx)
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to create pgx pool: %v", err)
	}
	defer pool.Close()

	// Repositories
	sourceRepo := repository.NewSourceRepository(db)
	listingRepo := repository.NewListingRepository(db)

	// Scraper engine with all scrapers registered
	eng := engine.NewEngine(sourceRepo, listingRepo)
	eng.RegisterScraper("bizbuysell", sources.NewBizBuySellScraper())
	eng.RegisterScraper("bizquest", sources.NewBizQuestScraper())
	eng.RegisterScraper("businessbroker", sources.NewBusinessBrokerScraper())
	eng.RegisterScraper("sunbelt", sources.NewSunbeltScraper())
	eng.RegisterScraper("transworld", sources.NewTransworldScraper())
	eng.RegisterScraper("firstchoice", sources.NewFirstChoiceScraper())

	// River workers
	workers := river.NewWorkers()
	river.AddWorker(workers, jobs.NewScrapeJobWorker(eng, sourceRepo, listingRepo))
	river.AddWorker(workers, jobs.NewScrapeAllJobWorker(eng, sourceRepo, listingRepo))

	// River client
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 2},
		},
		Workers:      workers,
		PeriodicJobs: jobs.GetPeriodicJobs(),
	})
	if err != nil {
		log.Fatalf("Failed to create River client: %v", err)
	}

	// Start the worker
	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("Failed to start River: %v", err)
	}

	log.Println("Scraper worker started. Waiting for jobs...")

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := riverClient.Stop(shutdownCtx); err != nil {
		log.Printf("Error stopping River: %v", err)
	}

	log.Println("Worker stopped")
}
