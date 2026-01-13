package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/google/uuid"

	"github.com/kbsch/trough/internal/domain"
	"github.com/kbsch/trough/internal/scraper/browser"
)

// BizBuySellRodScraper uses headless Chrome for scraping
type BizBuySellRodScraper struct {
	pool *browser.Pool
}

func NewBizBuySellRodScraper() (*BizBuySellRodScraper, error) {
	pool, err := browser.NewPool()
	if err != nil {
		return nil, fmt.Errorf("failed to create browser pool: %w", err)
	}
	return &BizBuySellRodScraper{pool: pool}, nil
}

func (s *BizBuySellRodScraper) Name() string {
	return "bizbuysell"
}

func (s *BizBuySellRodScraper) Close() error {
	if s.pool != nil {
		return s.pool.Close()
	}
	return nil
}

func (s *BizBuySellRodScraper) Scrape(ctx context.Context, opts domain.ScrapeOptions) (<-chan *domain.Listing, <-chan error) {
	listings := make(chan *domain.Listing, 100)
	errors := make(chan error, 10)

	go func() {
		defer close(listings)
		defer close(errors)

		page, err := s.pool.GetPage()
		if err != nil {
			errors <- fmt.Errorf("failed to get page: %w", err)
			return
		}
		defer page.Close()

		count := 0
		pageNum := 1
		maxPages := 50
		if opts.MaxListings > 0 {
			maxPages = (opts.MaxListings / 20) + 1
		}

		baseURL := "https://www.bizbuysell.com/businesses-for-sale/"

		for pageNum <= maxPages {
			var url string
			if pageNum == 1 {
				url = baseURL
			} else {
				url = fmt.Sprintf("%s%d/", baseURL, pageNum)
			}

			log.Printf("BizBuySell: scraping page %d: %s", pageNum, url)

			// Navigate to page
			if err := browser.NavigateWithRetry(page, url, 3); err != nil {
				errors <- fmt.Errorf("failed to navigate to page %d: %w", pageNum, err)
				break
			}

			// Wait for listings to load
			time.Sleep(2 * time.Second)

			// Check if we got blocked
			html, err := page.HTML()
			if err != nil {
				errors <- fmt.Errorf("failed to get HTML: %w", err)
				break
			}

			// Debug: log page title and part of HTML
			title := browser.GetText(page, "title")
			log.Printf("BizBuySell: page title: %s", title)

			htmlLower := strings.ToLower(html)
			if strings.Contains(htmlLower, "access denied") ||
			   strings.Contains(htmlLower, "captcha") ||
			   strings.Contains(htmlLower, "blocked") ||
			   strings.Contains(htmlLower, "cloudflare") ||
			   strings.Contains(htmlLower, "just a moment") {
				// Save debug info
				previewLen := 500
				if len(html) < previewLen {
					previewLen = len(html)
				}
				log.Printf("BizBuySell: blocked - HTML preview: %s", html[:previewLen])
				errors <- fmt.Errorf("access blocked on page %d (title: %s)", pageNum, title)
				break
			}

			// Scroll to load lazy content
			browser.ScrollToBottom(page)
			time.Sleep(1 * time.Second)

			// Parse listings
			pageListings, err := s.parseListingsFromPage(page)
			if err != nil {
				errors <- fmt.Errorf("failed to parse page %d: %w", pageNum, err)
				break
			}

			if len(pageListings) == 0 {
				log.Printf("BizBuySell: no listings found on page %d, stopping", pageNum)
				break
			}

			for _, listing := range pageListings {
				if opts.MaxListings > 0 && count >= opts.MaxListings {
					return
				}

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

			pageNum++

			// Random delay between pages
			browser.RandomDelay(2*time.Second, 5*time.Second)
		}

		log.Printf("BizBuySell: scrape completed with %d listings", count)
	}()

	return listings, errors
}

func (s *BizBuySellRodScraper) parseListingsFromPage(page *rod.Page) ([]*domain.Listing, error) {
	var listings []*domain.Listing

	// Find all listing cards - try multiple selectors
	selectors := []string{
		"div.listing",
		"div.diamond-listing",
		"article.listing",
		"div[class*='listing-card']",
		"div[class*='ListingCard']",
	}

	var elements rod.Elements
	for _, selector := range selectors {
		els, err := page.Elements(selector)
		if err == nil && len(els) > 0 {
			elements = els
			log.Printf("BizBuySell: found %d elements with selector: %s", len(els), selector)
			break
		}
	}

	if len(elements) == 0 {
		// Try to extract from page data/JSON
		return s.parseFromPageData(page)
	}

	for _, el := range elements {
		listing := s.parseListingElement(el)
		if listing != nil {
			listings = append(listings, listing)
		}
	}

	return listings, nil
}

func (s *BizBuySellRodScraper) parseListingElement(el *rod.Element) *domain.Listing {
	// Extract URL
	linkEl, err := el.Element("a")
	if err != nil {
		return nil
	}

	href, err := linkEl.Attribute("href")
	if err != nil || href == nil || *href == "" {
		return nil
	}

	url := *href
	if !strings.HasPrefix(url, "http") {
		url = "https://www.bizbuysell.com" + url
	}

	externalID := extractBizBuySellID(url)
	if externalID == "" {
		return nil
	}

	// Extract title
	title := ""
	titleSelectors := []string{"a.title", "h3 a", ".listing-title a", "a[class*='title']"}
	for _, sel := range titleSelectors {
		if titleEl, err := el.Element(sel); err == nil {
			if t, err := titleEl.Text(); err == nil && t != "" {
				title = strings.TrimSpace(t)
				break
			}
		}
	}

	if title == "" {
		// Fall back to any anchor text
		if t, err := linkEl.Text(); err == nil {
			title = strings.TrimSpace(t)
		}
	}

	if title == "" {
		return nil
	}

	listing := &domain.Listing{
		ID:         uuid.New(),
		ExternalID: externalID,
		URL:        url,
		Title:      title,
		Country:    domain.StrPtr("US"),
		IsActive:   true,
	}

	// Extract description
	descSelectors := []string{".description", ".listing-description", "p.desc", "p"}
	for _, sel := range descSelectors {
		if descEl, err := el.Element(sel); err == nil {
			if desc, err := descEl.Text(); err == nil && desc != "" {
				d := strings.TrimSpace(desc)
				listing.Description = &d
				break
			}
		}
	}

	// Extract price
	priceSelectors := []string{".price", ".asking-price", "[class*='price']", "span[data-price]"}
	for _, sel := range priceSelectors {
		if priceEl, err := el.Element(sel); err == nil {
			if priceText, err := priceEl.Text(); err == nil {
				if price := parsePrice(priceText); price > 0 {
					listing.AskingPrice = &price
					break
				}
			}
		}
	}

	// Extract cash flow
	cfSelectors := []string{".cash-flow", ".cashflow", "[class*='cashflow']"}
	for _, sel := range cfSelectors {
		if cfEl, err := el.Element(sel); err == nil {
			if cfText, err := cfEl.Text(); err == nil {
				if cf := parsePrice(cfText); cf > 0 {
					listing.CashFlow = &cf
					break
				}
			}
		}
	}

	// Extract location
	locSelectors := []string{".location", ".city-state", "[class*='location']"}
	for _, sel := range locSelectors {
		if locEl, err := el.Element(sel); err == nil {
			if locText, err := locEl.Text(); err == nil && locText != "" {
				city, state := parseLocation(locText)
				if city != "" {
					listing.City = &city
				}
				if state != "" {
					listing.State = &state
				}
				break
			}
		}
	}

	// Extract industry
	indSelectors := []string{".category", ".industry", "[class*='category']"}
	for _, sel := range indSelectors {
		if indEl, err := el.Element(sel); err == nil {
			if indText, err := indEl.Text(); err == nil && indText != "" {
				ind := strings.TrimSpace(indText)
				listing.Industry = &ind
				break
			}
		}
	}

	// Check for franchise/real estate keywords
	fullText, _ := el.Text()
	fullTextLower := strings.ToLower(fullText)

	if strings.Contains(fullTextLower, "franchise") {
		listing.IsFranchise = domain.BoolPtr(true)
	}
	if strings.Contains(fullTextLower, "real estate included") ||
	   strings.Contains(fullTextLower, "includes real estate") {
		listing.RealEstateIncluded = domain.BoolPtr(true)
	}

	// Store raw data
	rawData := map[string]interface{}{
		"source_url": url,
		"scraped_at": time.Now().Format(time.RFC3339),
		"method":     "rod",
	}
	if jsonBytes, err := json.Marshal(rawData); err == nil {
		listing.RawData = jsonBytes
	}

	return listing
}

func (s *BizBuySellRodScraper) parseFromPageData(page *rod.Page) ([]*domain.Listing, error) {
	// Try to find listing data in script tags or data attributes
	var listings []*domain.Listing

	// Look for JSON data in script tags
	scripts, err := page.Elements("script[type='application/ld+json']")
	if err == nil {
		for _, script := range scripts {
			content, err := script.Text()
			if err != nil {
				continue
			}

			// Try to parse as listing data
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(content), &data); err == nil {
				// Check if it's an ItemList with listings
				if items, ok := data["itemListElement"].([]interface{}); ok {
					for _, item := range items {
						if listing := s.parseJSONListing(item); listing != nil {
							listings = append(listings, listing)
						}
					}
				}
			}
		}
	}

	// Also try to extract from visible links/cards
	links, _ := page.Elements("a[href*='/Business-Opportunity/']")
	if len(links) > 0 {
		log.Printf("BizBuySell: found %d listing links", len(links))

		seenIDs := make(map[string]bool)
		for _, link := range links {
			href, err := link.Attribute("href")
			if err != nil || href == nil {
				continue
			}

			externalID := extractBizBuySellID(*href)
			if externalID == "" || seenIDs[externalID] {
				continue
			}
			seenIDs[externalID] = true

			title, _ := link.Text()
			title = strings.TrimSpace(title)
			if title == "" || len(title) < 5 {
				continue
			}

			url := *href
			if !strings.HasPrefix(url, "http") {
				url = "https://www.bizbuysell.com" + url
			}

			listing := &domain.Listing{
				ID:         uuid.New(),
				ExternalID: externalID,
				URL:        url,
				Title:      title,
				Country:    domain.StrPtr("US"),
				IsActive:   true,
			}
			listings = append(listings, listing)
		}
	}

	return listings, nil
}

func (s *BizBuySellRodScraper) parseJSONListing(item interface{}) *domain.Listing {
	data, ok := item.(map[string]interface{})
	if !ok {
		return nil
	}

	url, _ := data["url"].(string)
	if url == "" {
		return nil
	}

	name, _ := data["name"].(string)
	if name == "" {
		return nil
	}

	externalID := extractBizBuySellID(url)
	if externalID == "" {
		// Generate from URL
		re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
		externalID = re.ReplaceAllString(url, "-")
	}

	listing := &domain.Listing{
		ID:         uuid.New(),
		ExternalID: externalID,
		URL:        url,
		Title:      name,
		Country:    domain.StrPtr("US"),
		IsActive:   true,
	}

	if desc, ok := data["description"].(string); ok && desc != "" {
		listing.Description = &desc
	}

	return listing
}
