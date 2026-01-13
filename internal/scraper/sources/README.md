# Adding New Scrapers

This document describes how to add new business brokerage scrapers to Trough.

## Scraper Interface

All scrapers must implement the `Scraper` interface defined in `internal/scraper/engine/engine.go`:

```go
type Scraper interface {
    Name() string
    Scrape(ctx context.Context, opts domain.ScrapeOptions) (<-chan *domain.Listing, <-chan error)
}
```

## Creating a New Scraper

### 1. Create the Scraper File

Create a new file in `internal/scraper/sources/` named after the source (e.g., `newbroker.go`).

### 2. Implement the Scraper Struct

```go
package sources

import (
    "context"
    "github.com/kbsch/trough/internal/domain"
)

type NewBrokerScraper struct{}

func NewNewBrokerScraper() *NewBrokerScraper {
    return &NewBrokerScraper{}
}

func (s *NewBrokerScraper) Name() string {
    return "newbroker"
}

func (s *NewBrokerScraper) Scrape(ctx context.Context, opts domain.ScrapeOptions) (<-chan *domain.Listing, <-chan error) {
    listings := make(chan *domain.Listing, 100)
    errors := make(chan error, 10)

    go func() {
        defer close(listings)
        defer close(errors)

        // Scraping logic here
    }()

    return listings, errors
}
```

### 3. Scraping with Colly

Most scrapers use [Colly](https://github.com/gocolly/colly) for web scraping:

```go
c := colly.NewCollector(
    colly.AllowedDomains("www.example.com", "example.com"),
    colly.UserAgent("Mozilla/5.0 ..."),
    colly.MaxDepth(2),
)

// Rate limiting
c.Limit(&colly.LimitRule{
    DomainGlob:  "*example.com*",
    Delay:       opts.RateLimit,
    RandomDelay: 1 * time.Second,
    Parallelism: 1,
})

// Parse listings
c.OnHTML(".listing-card", func(e *colly.HTMLElement) {
    listing := parseListing(e)
    if listing != nil {
        listings <- listing
    }
})

// Handle pagination
c.OnHTML("a.next-page", func(e *colly.HTMLElement) {
    e.Request.Visit(e.Attr("href"))
})

// Start scraping
c.Visit("https://www.example.com/listings")
c.Wait()
```

### 4. Parsing Listings

Each listing should be parsed into a `domain.Listing` struct:

```go
listing := &domain.Listing{
    ID:         uuid.New(),
    ExternalID: extractID(url),        // Unique ID from source
    URL:        fullURL,                // Full URL to listing
    Title:      title,                  // Business name/title
    Description: description,           // Optional
    AskingPrice: &price,                // In cents
    Revenue:    &revenue,               // In cents
    CashFlow:   &cashFlow,              // In cents (SDE)
    City:       city,
    State:      state,
    Industry:   industry,
    IsFranchise: isFranchise,
    RealEstateIncluded: hasRealEstate,
    Country:    "US",
    IsActive:   true,
}
```

### 5. Helper Functions

Use the shared helper functions in `bizbuysell.go`:

- `parsePrice(text string) int64` - Parses price strings like "$500,000" to cents
- `parseLocation(text string) (city, state string)` - Parses "City, ST" format

### 6. Register the Scraper

Add the scraper to both entry points:

**cmd/cli/main.go** (in scrapeCmd function):
```go
eng.RegisterScraper("newbroker", sources.NewNewBrokerScraper())
```

**cmd/scraper/main.go** (scraper worker):
```go
eng.RegisterScraper("newbroker", sources.NewNewBrokerScraper())
```

### 7. Add to Seed Data

Add the source to the seed command in `cmd/cli/main.go`:

```go
{"New Broker", "newbroker", "https://www.newbroker.com", "colly"},
```

## Best Practices

1. **Rate Limiting**: Always use rate limiting (2+ seconds between requests)
2. **User Agent**: Use a realistic browser user agent
3. **Error Handling**: Send errors to the error channel, don't crash
4. **Context Cancellation**: Check `ctx.Done()` to support cancellation
5. **Deduplication**: Use consistent external IDs (prefix with source name)
6. **Multiple Selectors**: Try multiple CSS selectors for robustness
7. **Raw Data**: Store raw scraped data in `RawData` field for debugging

## Testing a Scraper

```bash
# Run with limit for testing
go run cmd/cli/main.go scrape run -s newbroker -l 10

# Check results
go run cmd/cli/main.go stats
```

## Current Scrapers

| Source | Slug | Type | URL |
|--------|------|------|-----|
| BizBuySell | bizbuysell | colly | bizbuysell.com |
| BizQuest | bizquest | colly | bizquest.com |
| BusinessBroker.net | businessbroker | colly | businessbroker.net |
| Sunbelt Network | sunbelt | colly | sunbeltnetwork.com |
| Transworld Business Advisors | transworld | colly | tworld.com |
| FirstChoice Business Brokers | firstchoice | colly | fcbb.com |
