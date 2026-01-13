package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"

	"github.com/kbsch/trough/internal/domain"
)

// TransworldScraper scrapes listings from Transworld Business Advisors
// A large national franchise business brokerage network
type TransworldScraper struct{}

func NewTransworldScraper() *TransworldScraper {
	return &TransworldScraper{}
}

func (s *TransworldScraper) Name() string {
	return "transworld"
}

func (s *TransworldScraper) Scrape(ctx context.Context, opts domain.ScrapeOptions) (<-chan *domain.Listing, <-chan error) {
	listings := make(chan *domain.Listing, 100)
	errors := make(chan error, 10)

	go func() {
		defer close(listings)
		defer close(errors)

		c := colly.NewCollector(
			colly.AllowedDomains("www.tworld.com", "tworld.com"),
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
			colly.MaxDepth(2),
		)

		c.Limit(&colly.LimitRule{
			DomainGlob:  "*tworld.com*",
			Delay:       opts.RateLimit,
			RandomDelay: 1 * time.Second,
			Parallelism: 1,
		})

		count := 0
		pageCount := 0
		maxPages := 50
		if opts.MaxListings > 0 {
			maxPages = (opts.MaxListings / 20) + 1
		}

		// Parse listing cards from search results
		c.OnHTML(".listing-card, .business-listing, .listing-row, .listing-item, article.business", func(e *colly.HTMLElement) {
			if opts.MaxListings > 0 && count >= opts.MaxListings {
				return
			}

			listing := s.parseListingCard(e)
			if listing != nil {
				select {
				case listings <- listing:
					count++
					if count%10 == 0 {
						log.Printf("Transworld: scraped %d listings", count)
					}
				case <-ctx.Done():
					return
				}
			}
		})

		// Alternative selector for card-based layouts
		c.OnHTML(".business-card, div[data-business-id]", func(e *colly.HTMLElement) {
			if opts.MaxListings > 0 && count >= opts.MaxListings {
				return
			}

			listing := s.parseBusinessCard(e)
			if listing != nil {
				select {
				case listings <- listing:
					count++
				case <-ctx.Done():
					return
				}
			}
		})

		// Follow pagination
		c.OnHTML("a.next-page, a[rel='next'], .pagination a.next, .pager a.next", func(e *colly.HTMLElement) {
			if opts.MaxListings > 0 && count >= opts.MaxListings {
				return
			}
			if pageCount >= maxPages {
				return
			}

			nextURL := e.Attr("href")
			if nextURL != "" && !strings.HasPrefix(nextURL, "javascript:") && !strings.Contains(e.Attr("class"), "disabled") {
				if !strings.HasPrefix(nextURL, "http") {
					nextURL = "https://www.tworld.com" + nextURL
				}
				pageCount++
				log.Printf("Transworld: following page %d: %s", pageCount, nextURL)
				e.Request.Visit(nextURL)
			}
		})

		c.OnError(func(r *colly.Response, err error) {
			select {
			case errors <- fmt.Errorf("request error %d: %s - %v", r.StatusCode, r.Request.URL, err):
			default:
			}
		})

		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
			r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
			r.Headers.Set("Connection", "keep-alive")
		})

		startURL := "https://www.tworld.com/businesses-for-sale/"
		log.Printf("Transworld: starting scrape from %s", startURL)

		if err := c.Visit(startURL); err != nil {
			errors <- fmt.Errorf("failed to start scrape: %w", err)
		}

		c.Wait()
		log.Printf("Transworld: scrape completed with %d listings", count)
	}()

	return listings, errors
}

func (s *TransworldScraper) parseListingCard(e *colly.HTMLElement) *domain.Listing {
	// Try multiple selectors for URL
	url := e.ChildAttr("a.listing-title", "href")
	if url == "" {
		url = e.ChildAttr("h3 a", "href")
	}
	if url == "" {
		url = e.ChildAttr("a.title", "href")
	}
	if url == "" {
		url = e.ChildAttr("a[href*='/listing/']", "href")
	}
	if url == "" {
		url = e.ChildAttr("a[href*='/business/']", "href")
	}
	if url == "" {
		url = e.ChildAttr("a", "href")
	}
	if url == "" {
		return nil
	}

	externalID := extractTransworldID(url)
	if externalID == "" {
		return nil
	}

	// Parse title
	title := strings.TrimSpace(e.ChildText("a.listing-title"))
	if title == "" {
		title = strings.TrimSpace(e.ChildText("h3 a"))
	}
	if title == "" {
		title = strings.TrimSpace(e.ChildText(".listing-title"))
	}
	if title == "" {
		title = strings.TrimSpace(e.ChildText("h4 a"))
	}
	if title == "" {
		title = strings.TrimSpace(e.ChildText(".business-name"))
	}
	if title == "" {
		return nil
	}

	fullURL := url
	if !strings.HasPrefix(url, "http") {
		fullURL = "https://www.tworld.com" + url
	}

	listing := &domain.Listing{
		ID:         uuid.New(),
		ExternalID: externalID,
		URL:        fullURL,
		Title:      title,
		Country:    domain.StrPtr("US"),
		IsActive:   true,
	}

	// Parse description
	desc := strings.TrimSpace(e.ChildText(".listing-description, .description, p.summary, .business-description"))
	if desc != "" {
		listing.Description = &desc
	}

	// Parse asking price
	priceText := e.ChildText(".asking-price, .price, .listing-price")
	if price := parsePrice(priceText); price > 0 {
		listing.AskingPrice = &price
	}

	// Parse cash flow
	cashFlowText := e.ChildText(".cash-flow, .cashflow, .sde, .net-income")
	if cf := parsePrice(cashFlowText); cf > 0 {
		listing.CashFlow = &cf
	}

	// Parse revenue
	revenueText := e.ChildText(".revenue, .gross-revenue, .gross-sales, .annual-revenue")
	if rev := parsePrice(revenueText); rev > 0 {
		listing.Revenue = &rev
	}

	// Parse location
	location := strings.TrimSpace(e.ChildText(".location, .city-state, .listing-location, .business-location"))
	if location != "" {
		city, state := parseLocation(location)
		if city != "" {
			listing.City = &city
		}
		if state != "" {
			listing.State = &state
		}
	}

	// Parse industry
	industry := strings.TrimSpace(e.ChildText(".category, .industry, .business-type, .business-category"))
	if industry != "" {
		listing.Industry = &industry
	}

	// Check for franchise
	if strings.Contains(strings.ToLower(e.Text), "franchise") {
		listing.IsFranchise = domain.BoolPtr(true)
	}

	// Check for real estate
	if strings.Contains(strings.ToLower(e.Text), "real estate included") ||
		strings.Contains(strings.ToLower(e.Text), "includes real estate") {
		listing.RealEstateIncluded = domain.BoolPtr(true)
	}

	rawData := map[string]interface{}{
		"source_url": url,
		"scraped_at": time.Now().Format(time.RFC3339),
	}
	if jsonBytes, err := json.Marshal(rawData); err == nil {
		listing.RawData = jsonBytes
	}

	return listing
}

func (s *TransworldScraper) parseBusinessCard(e *colly.HTMLElement) *domain.Listing {
	listingID := e.Attr("data-business-id")
	if listingID == "" {
		listingID = e.Attr("data-listing-id")
	}
	if listingID == "" {
		listingID = e.Attr("data-id")
	}
	if listingID == "" {
		return nil
	}

	url := e.ChildAttr("a", "href")
	if url == "" {
		return nil
	}

	title := strings.TrimSpace(e.ChildText("h3, h4, .title, .business-name"))
	if title == "" {
		return nil
	}

	fullURL := url
	if !strings.HasPrefix(url, "http") {
		fullURL = "https://www.tworld.com" + url
	}

	listing := &domain.Listing{
		ID:         uuid.New(),
		ExternalID: "tw-" + listingID,
		URL:        fullURL,
		Title:      title,
		Country:    domain.StrPtr("US"),
		IsActive:   true,
	}

	// Parse data attributes
	if price := e.Attr("data-price"); price != "" {
		if p := parsePrice(price); p > 0 {
			listing.AskingPrice = &p
		}
	}

	if loc := e.Attr("data-location"); loc != "" {
		city, state := parseLocation(loc)
		if city != "" {
			listing.City = &city
		}
		if state != "" {
			listing.State = &state
		}
	}

	if industry := e.Attr("data-category"); industry != "" {
		listing.Industry = &industry
	}

	return listing
}

func extractTransworldID(url string) string {
	patterns := []string{
		`/listing/(\d+)`,
		`/business/(\d+)`,
		`/(\d+)/?$`,
		`id=(\d+)`,
		`listing-(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) >= 2 {
			return "tw-" + matches[1]
		}
	}

	// Fallback: use URL slug as ID
	re := regexp.MustCompile(`/([a-z0-9-]+)/?$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) >= 2 && matches[1] != "" && matches[1] != "businesses-for-sale" {
		return "tw-" + matches[1]
	}

	return ""
}
