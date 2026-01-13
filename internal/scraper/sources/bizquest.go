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

type BizQuestScraper struct{}

func NewBizQuestScraper() *BizQuestScraper {
	return &BizQuestScraper{}
}

func (s *BizQuestScraper) Name() string {
	return "bizquest"
}

func (s *BizQuestScraper) Scrape(ctx context.Context, opts domain.ScrapeOptions) (<-chan *domain.Listing, <-chan error) {
	listings := make(chan *domain.Listing, 100)
	errors := make(chan error, 10)

	go func() {
		defer close(listings)
		defer close(errors)

		c := colly.NewCollector(
			colly.AllowedDomains("www.bizquest.com", "bizquest.com"),
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
			colly.MaxDepth(2),
		)

		c.Limit(&colly.LimitRule{
			DomainGlob:  "*bizquest.com*",
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

		// BizQuest listing cards
		c.OnHTML("div.listing-item, article.listing, div.search-result-item", func(e *colly.HTMLElement) {
			if opts.MaxListings > 0 && count >= opts.MaxListings {
				return
			}

			listing := s.parseListingCard(e)
			if listing != nil {
				select {
				case listings <- listing:
					count++
					if count%10 == 0 {
						log.Printf("BizQuest: scraped %d listings", count)
					}
				case <-ctx.Done():
					return
				}
			}
		})

		// Pagination
		c.OnHTML("a.next, a[rel='next'], .pagination a:last-child", func(e *colly.HTMLElement) {
			if opts.MaxListings > 0 && count >= opts.MaxListings {
				return
			}
			if pageCount >= maxPages {
				return
			}

			nextURL := e.Attr("href")
			if nextURL != "" && !strings.HasPrefix(nextURL, "javascript:") && !strings.Contains(e.Text, "Previous") {
				pageCount++
				if !strings.HasPrefix(nextURL, "http") {
					nextURL = "https://www.bizquest.com" + nextURL
				}
				log.Printf("BizQuest: following page %d", pageCount)
				e.Request.Visit(nextURL)
			}
		})

		c.OnError(func(r *colly.Response, err error) {
			select {
			case errors <- fmt.Errorf("BizQuest request error %d: %s - %v", r.StatusCode, r.Request.URL, err):
			default:
			}
		})

		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
			r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		})

		startURL := "https://www.bizquest.com/businesses-for-sale/"
		log.Printf("BizQuest: starting scrape from %s", startURL)

		if err := c.Visit(startURL); err != nil {
			errors <- fmt.Errorf("BizQuest failed to start: %w", err)
		}

		c.Wait()
		log.Printf("BizQuest: scrape completed with %d listings", count)
	}()

	return listings, errors
}

func (s *BizQuestScraper) parseListingCard(e *colly.HTMLElement) *domain.Listing {
	// Try to find the listing URL
	url := e.ChildAttr("a.listing-title", "href")
	if url == "" {
		url = e.ChildAttr("h3 a", "href")
	}
	if url == "" {
		url = e.ChildAttr("a[href*='/business-for-sale/']", "href")
	}
	if url == "" {
		return nil
	}

	externalID := extractBizQuestID(url)
	if externalID == "" {
		return nil
	}

	title := strings.TrimSpace(e.ChildText("a.listing-title, h3 a, .listing-title"))
	if title == "" {
		return nil
	}

	fullURL := url
	if !strings.HasPrefix(url, "http") {
		fullURL = "https://www.bizquest.com" + url
	}

	listing := &domain.Listing{
		ID:         uuid.New(),
		ExternalID: externalID,
		URL:        fullURL,
		Title:      title,
		Country:    domain.StrPtr("US"),
		IsActive:   true,
	}

	// Description
	if desc := strings.TrimSpace(e.ChildText(".listing-description, .description, p")); desc != "" {
		listing.Description = &desc
	}

	// Price
	priceText := e.ChildText(".price, .asking-price, .listing-price")
	if price := parsePrice(priceText); price > 0 {
		listing.AskingPrice = &price
	}

	// Cash flow
	cfText := e.ChildText(".cash-flow, .cashflow")
	if cf := parsePrice(cfText); cf > 0 {
		listing.CashFlow = &cf
	}

	// Revenue
	revText := e.ChildText(".revenue, .gross-revenue")
	if rev := parsePrice(revText); rev > 0 {
		listing.Revenue = &rev
	}

	// Location
	location := strings.TrimSpace(e.ChildText(".location, .city-state"))
	if location != "" {
		city, state := parseLocation(location)
		if city != "" {
			listing.City = &city
		}
		if state != "" {
			listing.State = &state
		}
	}

	// Industry
	if industry := strings.TrimSpace(e.ChildText(".category, .industry")); industry != "" {
		listing.Industry = &industry
	}

	// Franchise check
	if strings.Contains(strings.ToLower(e.Text), "franchise") {
		listing.IsFranchise = domain.BoolPtr(true)
	}

	// Real estate check
	if strings.Contains(strings.ToLower(e.Text), "real estate") {
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

func extractBizQuestID(url string) string {
	// BizQuest URL formats:
	// /business-for-sale/detail/123456/
	// /listing/123456
	patterns := []string{
		`/detail/(\d+)`,
		`/listing/(\d+)`,
		`-(\d+)/?$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	// Fallback: hash the URL
	return ""
}
