package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/spf13/cobra"

	"github.com/kbsch/trough/internal/domain"
	"github.com/kbsch/trough/internal/repository"
	"github.com/kbsch/trough/internal/scraper/engine"
	"github.com/kbsch/trough/internal/scraper/jobs"
	"github.com/kbsch/trough/internal/scraper/sources"
)

var (
	db     *sqlx.DB
	dbURL  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "trough",
		Short: "Trough - Business Broker Listing Aggregator CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip DB connection for help commands
			if cmd.Name() == "help" || cmd.Name() == "version" {
				return nil
			}

			dbURL = os.Getenv("DATABASE_URL")
			if dbURL == "" {
				dbURL = "postgres://trough:trough@localhost:5432/trough?sslmode=disable"
			}

			var err error
			db, err = sqlx.Connect("postgres", dbURL)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if db != nil {
				db.Close()
			}
		},
	}

	rootCmd.AddCommand(scrapeCmd())
	rootCmd.AddCommand(seedCmd())
	rootCmd.AddCommand(queueCmd())
	rootCmd.AddCommand(statsCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func scrapeCmd() *cobra.Command {
	var sourceSlug string
	var limit int

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Run scrapers directly (not via job queue)",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run a scraper for a specific source or all sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			sourceRepo := repository.NewSourceRepository(db)
			listingRepo := repository.NewListingRepository(db)

			eng := engine.NewEngine(sourceRepo, listingRepo)
			eng.RegisterScraper("bizbuysell", sources.NewBizBuySellScraper())
			eng.RegisterScraper("bizquest", sources.NewBizQuestScraper())
			eng.RegisterScraper("businessbroker", sources.NewBusinessBrokerScraper())
			eng.RegisterScraper("sunbelt", sources.NewSunbeltScraper())
			eng.RegisterScraper("transworld", sources.NewTransworldScraper())
			eng.RegisterScraper("firstchoice", sources.NewFirstChoiceScraper())

			if sourceSlug == "" {
				log.Println("Running all active scrapers...")
				return eng.RunAll(ctx)
			}

			log.Printf("Running scraper for: %s", sourceSlug)
			return eng.RunSource(ctx, sourceSlug, limit)
		},
	}
	runCmd.Flags().StringVarP(&sourceSlug, "source", "s", "", "Source slug to scrape (empty for all)")
	runCmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of listings (0 for unlimited)")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available scrapers and sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			sourceRepo := repository.NewSourceRepository(db)

			sources, err := sourceRepo.ListActive(ctx)
			if err != nil {
				return fmt.Errorf("failed to list sources: %w", err)
			}

			fmt.Println("Available sources:")
			fmt.Println("------------------")
			for _, s := range sources {
				status := "active"
				if !s.IsActive {
					status = "inactive"
				}
				fmt.Printf("  %s (%s) - %s [%s]\n", s.Name, s.Slug, s.BaseURL, status)
			}

			return nil
		},
	}

	cmd.AddCommand(runCmd)
	cmd.AddCommand(listCmd)
	return cmd
}

func seedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with initial sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			sourceRepo := repository.NewSourceRepository(db)

			sources := []struct {
				name        string
				slug        string
				baseURL     string
				scraperType string
			}{
				{"BizBuySell", "bizbuysell", "https://www.bizbuysell.com", "colly"},
				{"BizQuest", "bizquest", "https://www.bizquest.com", "colly"},
				{"BusinessBroker.net", "businessbroker", "https://www.businessbroker.net", "colly"},
				{"Sunbelt Network", "sunbelt", "https://www.sunbeltnetwork.com", "colly"},
				{"Transworld Business Advisors", "transworld", "https://www.tworld.com", "colly"},
				{"FirstChoice Business Brokers", "firstchoice", "https://www.fcbb.com", "colly"},
			}

			for _, s := range sources {
				source := &domain.Source{
					ID:          uuid.New(),
					Name:        s.name,
					Slug:        s.slug,
					BaseURL:     s.baseURL,
					ScraperType: s.scraperType,
					IsActive:    true,
					Config:      []byte("{}"),
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}

				if err := sourceRepo.Create(ctx, source); err != nil {
					if !strings.Contains(err.Error(), "duplicate key") {
						log.Printf("Warning: failed to seed source %s: %v", s.name, err)
					} else {
						log.Printf("Source already exists: %s", s.name)
					}
				} else {
					log.Printf("Seeded source: %s", s.name)
				}
			}

			return nil
		},
	}
}

func queueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Manage the job queue",
	}

	var sourceSlug string
	var maxListings int

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a scrape job to the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			pool, err := pgxpool.New(ctx, dbURL)
			if err != nil {
				return fmt.Errorf("failed to create pgx pool: %w", err)
			}
			defer pool.Close()

			client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
			if err != nil {
				return fmt.Errorf("failed to create River client: %w", err)
			}

			if sourceSlug == "" {
				// Queue all sources
				result, err := client.Insert(ctx, jobs.ScrapeAllJobArgs{}, nil)
				if err != nil {
					return fmt.Errorf("failed to insert job: %w", err)
				}
				log.Printf("Queued scrape-all job: %d", result.Job.ID)
			} else {
				result, err := client.Insert(ctx, jobs.ScrapeJobArgs{
					SourceSlug:  sourceSlug,
					MaxListings: maxListings,
					FullScrape:  true,
				}, nil)
				if err != nil {
					return fmt.Errorf("failed to insert job: %w", err)
				}
				log.Printf("Queued scrape job for %s: %d", sourceSlug, result.Job.ID)
			}

			return nil
		},
	}
	addCmd.Flags().StringVarP(&sourceSlug, "source", "s", "", "Source slug (empty for all)")
	addCmd.Flags().IntVarP(&maxListings, "limit", "l", 0, "Max listings (0 for unlimited)")

	cmd.AddCommand(addCmd)
	return cmd
}

func statsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show database statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			var listingCount int
			if err := db.GetContext(ctx, &listingCount, "SELECT COUNT(*) FROM listings WHERE is_active = true"); err != nil {
				return fmt.Errorf("failed to count listings: %w", err)
			}

			var sourceCount int
			if err := db.GetContext(ctx, &sourceCount, "SELECT COUNT(*) FROM sources WHERE is_active = true"); err != nil {
				return fmt.Errorf("failed to count sources: %w", err)
			}

			type sourceStat struct {
				Name  string `db:"name"`
				Count int    `db:"count"`
			}
			var bySource []sourceStat
			err := db.SelectContext(ctx, &bySource, `
				SELECT s.name, COUNT(l.id) as count
				FROM sources s
				LEFT JOIN listings l ON l.source_id = s.id AND l.is_active = true
				WHERE s.is_active = true
				GROUP BY s.id, s.name
				ORDER BY count DESC
			`)
			if err != nil {
				return fmt.Errorf("failed to get source stats: %w", err)
			}

			fmt.Println("Trough Statistics")
			fmt.Println("=================")
			fmt.Printf("Active listings: %d\n", listingCount)
			fmt.Printf("Active sources: %d\n", sourceCount)
			fmt.Println()
			fmt.Println("Listings by source:")
			for _, s := range bySource {
				fmt.Printf("  %s: %d\n", s.Name, s.Count)
			}

			return nil
		},
	}
}
