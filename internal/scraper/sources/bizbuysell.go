package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"

	"github.com/kbsch/trough/internal/domain"
)

type BizBuySellScraper struct{}

func NewBizBuySellScraper() *BizBuySellScraper {
	return &BizBuySellScraper{}
}

func (s *BizBuySellScraper) Name() string {
	return "bizbuysell"
}

func (s *BizBuySellScraper) Scrape(ctx context.Context, opts domain.ScrapeOptions) (<-chan *domain.Listing, <-chan error) {
	listings := make(chan *domain.Listing, 100)
	errors := make(chan error, 10)

	go func() {
		defer close(listings)
		defer close(errors)

		c := colly.NewCollector(
			colly.AllowedDomains("www.bizbuysell.com", "bizbuysell.com"),
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
			colly.MaxDepth(2),
		)

		c.Limit(&colly.LimitRule{
			DomainGlob:  "*bizbuysell.com*",
			Delay:       opts.RateLimit,
			RandomDelay: 1 * time.Second,
			Parallelism: 1,
		})

		count := 0
		pageCount := 0
		maxPages := 50 // Default max pages
		if opts.MaxListings > 0 {
			maxPages = (opts.MaxListings / 20) + 1
		}

		// Parse listing cards from search results
		// BizBuySell uses .listing-card or similar for each listing
		c.OnHTML("div.listing, div.listing-card, article.listing", func(e *colly.HTMLElement) {
			if opts.MaxListings > 0 && count >= opts.MaxListings {
				return
			}

			listing := s.parseListingCard(e)
			if listing != nil {
				select {
				case listings <- listing:
					count++
					if count%10 == 0 {
						log.Printf("BizBuySell: scraped %d listings", count)
					}
				case <-ctx.Done():
					return
				}
			}
		})

		// Alternative selector for newer BizBuySell layout
		c.OnHTML("div[data-listing-id]", func(e *colly.HTMLElement) {
			if opts.MaxListings > 0 && count >= opts.MaxListings {
				return
			}

			listing := s.parseDataListing(e)
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
		c.OnHTML("a.next, a[rel='next'], .pagination a:contains('Next')", func(e *colly.HTMLElement) {
			if opts.MaxListings > 0 && count >= opts.MaxListings {
				return
			}
			if pageCount >= maxPages {
				return
			}
			pageCount++

			nextURL := e.Attr("href")
			if nextURL != "" && !strings.HasPrefix(nextURL, "javascript:") {
				if !strings.HasPrefix(nextURL, "http") {
					nextURL = "https://www.bizbuysell.com" + nextURL
				}
				log.Printf("BizBuySell: following page %d: %s", pageCount, nextURL)
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
			// Add headers to appear more like a browser
			r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
			r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
			r.Headers.Set("Connection", "keep-alive")
		})

		// Start with main search page
		startURL := "https://www.bizbuysell.com/businesses-for-sale/"
		log.Printf("BizBuySell: starting scrape from %s", startURL)

		if err := c.Visit(startURL); err != nil {
			errors <- fmt.Errorf("failed to start scrape: %w", err)
		}

		c.Wait()
		log.Printf("BizBuySell: scrape completed with %d listings", count)
	}()

	return listings, errors
}

func (s *BizBuySellScraper) parseListingCard(e *colly.HTMLElement) *domain.Listing {
	// Try multiple selectors for the URL
	url := e.ChildAttr("a.title", "href")
	if url == "" {
		url = e.ChildAttr("a.listing-title", "href")
	}
	if url == "" {
		url = e.ChildAttr("h3 a", "href")
	}
	if url == "" {
		url = e.ChildAttr("a[href*='/Business-Opportunity/']", "href")
	}
	if url == "" {
		return nil
	}

	externalID := extractBizBuySellID(url)
	if externalID == "" {
		return nil
	}

	// Try multiple selectors for title
	title := strings.TrimSpace(e.ChildText("a.title"))
	if title == "" {
		title = strings.TrimSpace(e.ChildText("a.listing-title"))
	}
	if title == "" {
		title = strings.TrimSpace(e.ChildText("h3 a"))
	}
	if title == "" {
		title = strings.TrimSpace(e.ChildText(".listing-title"))
	}
	if title == "" {
		return nil
	}

	fullURL := url
	if !strings.HasPrefix(url, "http") {
		fullURL = "https://www.bizbuysell.com" + url
	}

	listing := &domain.Listing{
		ID:         uuid.New(),
		ExternalID: externalID,
		URL:        fullURL,
		Title:      title,
		Country:    "US",
		IsActive:   true,
	}

	// Parse description
	desc := strings.TrimSpace(e.ChildText(".listing-description, .description, p.desc"))
	if desc != "" {
		listing.Description = desc
	}

	// Parse price - try multiple selectors
	priceText := e.ChildText(".price, .asking-price, .listing-price, span[data-price]")
	if price := parsePrice(priceText); price > 0 {
		listing.AskingPrice = &price
	}

	// Parse cash flow
	cashFlowText := e.ChildText(".cash-flow, .cashflow, [data-cashflow]")
	if cf := parsePrice(cashFlowText); cf > 0 {
		listing.CashFlow = &cf
	}

	// Parse revenue
	revenueText := e.ChildText(".revenue, .gross-revenue, [data-revenue]")
	if rev := parsePrice(revenueText); rev > 0 {
		listing.Revenue = &rev
	}

	// Parse location
	location := strings.TrimSpace(e.ChildText(".location, .listing-location, .city-state"))
	if location != "" {
		city, state := parseLocation(location)
		listing.City = city
		listing.State = state
	}

	// Parse industry/category
	industry := strings.TrimSpace(e.ChildText(".category, .industry, .listing-category"))
	if industry != "" {
		listing.Industry = industry
	}

	// Check for franchise
	if strings.Contains(strings.ToLower(e.Text), "franchise") {
		listing.IsFranchise = true
	}

	// Check for real estate
	if strings.Contains(strings.ToLower(e.Text), "real estate included") ||
		strings.Contains(strings.ToLower(e.Text), "includes real estate") {
		listing.RealEstateIncluded = true
	}

	// Store raw HTML for debugging
	rawData := map[string]interface{}{
		"source_url": url,
		"scraped_at": time.Now().Format(time.RFC3339),
	}
	if jsonBytes, err := json.Marshal(rawData); err == nil {
		listing.RawData = jsonBytes
	}

	return listing
}

func (s *BizBuySellScraper) parseDataListing(e *colly.HTMLElement) *domain.Listing {
	listingID := e.Attr("data-listing-id")
	if listingID == "" {
		return nil
	}

	url := e.ChildAttr("a", "href")
	if url == "" {
		return nil
	}

	title := strings.TrimSpace(e.ChildText("h3, h4, .title"))
	if title == "" {
		return nil
	}

	fullURL := url
	if !strings.HasPrefix(url, "http") {
		fullURL = "https://www.bizbuysell.com" + url
	}

	listing := &domain.Listing{
		ID:         uuid.New(),
		ExternalID: listingID,
		URL:        fullURL,
		Title:      title,
		Country:    "US",
		IsActive:   true,
	}

	// Parse other fields from data attributes if available
	if price := e.Attr("data-price"); price != "" {
		if p := parsePrice(price); p > 0 {
			listing.AskingPrice = &p
		}
	}

	if cashflow := e.Attr("data-cashflow"); cashflow != "" {
		if cf := parsePrice(cashflow); cf > 0 {
			listing.CashFlow = &cf
		}
	}

	if loc := e.Attr("data-location"); loc != "" {
		city, state := parseLocation(loc)
		listing.City = city
		listing.State = state
	}

	if industry := e.Attr("data-category"); industry != "" {
		listing.Industry = industry
	}

	return listing
}

func extractBizBuySellID(url string) string {
	// URL formats:
	// /Business-Opportunity/listing-123456.aspx
	// /buy/listing-123456
	// /-123456.aspx
	patterns := []string{
		`listing-(\d+)`,
		`-(\d+)\.aspx`,
		`/(\d+)$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) >= 2 {
			return matches[1]
		}
	}
	return ""
}

func parsePrice(text string) int64 {
	if text == "" {
		return 0
	}

	// Remove currency symbols, commas, whitespace, and common words
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "$", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "asking price", "")
	text = strings.ReplaceAll(text, "cash flow", "")
	text = strings.ReplaceAll(text, "revenue", "")
	text = strings.TrimSpace(text)

	// Handle ranges like "$100,000 - $200,000" - take the first value
	if strings.Contains(text, "-") {
		parts := strings.Split(text, "-")
		text = strings.TrimSpace(parts[0])
	}

	// Handle "not disclosed", "call", etc.
	if strings.Contains(text, "disclosed") || strings.Contains(text, "call") ||
		strings.Contains(text, "contact") || strings.Contains(text, "n/a") {
		return 0
	}

	// Extract first number found
	re := regexp.MustCompile(`[\d.]+`)
	match := re.FindString(text)
	if match == "" {
		return 0
	}

	val, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0
	}

	// Handle millions/thousands abbreviations
	if strings.Contains(text, "m") || strings.Contains(text, "mil") {
		val *= 1000000
	} else if strings.Contains(text, "k") {
		val *= 1000
	}

	// Convert to cents
	return int64(val * 100)
}

func parseLocation(text string) (city, state string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}

	// Common format: "City, ST" or "City, State"
	parts := strings.Split(text, ",")
	if len(parts) >= 2 {
		city = strings.TrimSpace(parts[0])
		state = strings.TrimSpace(parts[1])
		// Clean up state - might have extra text
		state = strings.Split(state, " ")[0]
		state = strings.ToUpper(state)
	} else {
		// Might just be a state abbreviation
		if len(text) == 2 {
			state = strings.ToUpper(text)
		}
	}

	return city, state
}
