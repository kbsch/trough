# Trough

A nationwide business-for-sale listing aggregator that scrapes multiple brokerage sources and presents them through a unified search portal with faceted filters and map view.

## Features

- **Unified Search**: Search across multiple business listing sources
- **Faceted Filters**: Filter by price, location, industry, franchise status, etc.
- **Map View**: Visualize listings on an interactive Google Map
- **Real-time Data**: Daily automated scraping + on-demand refresh
- **RESTful API**: Full API access for integrations
- **Prometheus Metrics**: Built-in observability

## Tech Stack

- **Backend**: Go 1.22+ with Chi router
- **Database**: PostgreSQL 16 with PostGIS
- **Frontend**: SvelteKit 2.x with TypeScript
- **Scraping**: Colly (Go web scraping framework)
- **Queue**: River (PostgreSQL-backed job queue)
- **Containerization**: Docker + Docker Compose

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- PostgreSQL 16 with PostGIS
- Docker (optional)

### Development Setup

```bash
# Clone repository
git clone <repo-url>
cd trough

# Start PostgreSQL (with Docker)
docker run -d --name trough-postgres \
  -e POSTGRES_USER=trough \
  -e POSTGRES_PASSWORD=trough \
  -e POSTGRES_DB=trough \
  -p 5432:5432 \
  postgis/postgis:16-3.4

# Run migrations
go run cmd/cli/main.go migrate up

# Seed initial sources
go run cmd/cli/main.go seed

# Start API server
go run cmd/api/main.go

# In another terminal, start frontend
cd web
npm install
npm run dev
```

Visit http://localhost:5173 to see the application.

### Docker Compose (Development)

```bash
docker compose up -d
```

### Docker Compose (Production)

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for production deployment guide.

```bash
# Configure environment
cp .env.production.example .env.production
vim .env.production

# Deploy
docker compose -f docker-compose.prod.yml up -d
```

## Project Structure

```
trough/
├── cmd/
│   ├── api/           # API server entry point
│   ├── scraper/       # Scraper worker entry point
│   └── cli/           # Admin CLI tools
├── internal/
│   ├── api/           # HTTP handlers, middleware, routes
│   ├── domain/        # Core business types
│   ├── repository/    # Database access layer
│   └── scraper/       # Scraping engine + sources
├── migrations/        # SQL migrations
├── web/               # SvelteKit frontend
├── nginx/             # Nginx configuration
├── docs/              # Documentation
└── docker-compose.yml
```

## Data Sources

| Source | Type | Status |
|--------|------|--------|
| BizBuySell | Aggregator | Active |
| BizQuest | Aggregator | Active |
| BusinessBroker.net | Aggregator | Active |
| Sunbelt Network | Brokerage | Active |
| Transworld Business Advisors | Brokerage | Active |
| FirstChoice Business Brokers | Brokerage | Active |

See [internal/scraper/sources/README.md](internal/scraper/sources/README.md) for adding new sources.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/metrics` | Prometheus metrics |
| GET | `/api/v1/listings` | Search listings |
| GET | `/api/v1/listings/:id` | Get listing by ID |
| GET | `/api/v1/listings/map` | Get map markers |
| GET | `/api/v1/filters` | Get filter options |
| GET | `/api/v1/sources` | List active sources |
| POST | `/api/v1/refresh` | Trigger on-demand scrape |
| GET | `/api/v1/scrape-jobs` | Get scrape job history |

### Search Parameters

```
GET /api/v1/listings?q=restaurant&state=CA,TX&price_max=500000
```

| Parameter | Description |
|-----------|-------------|
| `q` | Full-text search query |
| `price_min`, `price_max` | Price range (in cents) |
| `revenue_min` | Minimum revenue |
| `cash_flow_min` | Minimum cash flow |
| `state` | States (comma-separated) |
| `industry` | Industries (comma-separated) |
| `franchise` | Franchise only (true/false) |
| `real_estate` | Includes real estate (true/false) |
| `bounds` | Map bounds (south,west,north,east) |
| `sort` | Sort order (price_asc, price_desc, newest) |
| `page`, `per_page` | Pagination |

## CLI Commands

```bash
# Run database migrations
go run cmd/cli/main.go migrate up

# Seed initial sources
go run cmd/cli/main.go seed

# Run scrapers
go run cmd/cli/main.go scrape run                    # All sources
go run cmd/cli/main.go scrape run -s bizbuysell -l 50  # Specific source, limit 50

# List available scrapers
go run cmd/cli/main.go scrape list

# View statistics
go run cmd/cli/main.go stats

# Queue a scrape job
go run cmd/cli/main.go queue add -s bizbuysell
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://trough:trough@localhost:5432/trough?sslmode=disable` |
| `PORT` | API server port | `8080` |
| `PUBLIC_API_URL` | Frontend API URL | `http://localhost:8080` |
| `PUBLIC_GOOGLE_MAPS_API_KEY` | Google Maps API key | - |

## Development

### Running Tests

```bash
go test ./...
```

### Adding a New Scraper

1. Create scraper in `internal/scraper/sources/`
2. Register in `cmd/cli/main.go` and `cmd/scraper/main.go`
3. Add to seed data in `cmd/cli/main.go`

See [internal/scraper/sources/README.md](internal/scraper/sources/README.md) for details.

### Frontend Development

```bash
cd web
npm run dev      # Development server
npm run build    # Production build
npm run check    # Type checking
```

## License

MIT
