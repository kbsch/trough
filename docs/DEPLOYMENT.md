# Trough Deployment Guide

This guide covers deploying Trough to production environments.

## Prerequisites

- Docker and Docker Compose
- A server with at least 2GB RAM
- (Optional) Domain name and SSL certificate for HTTPS

## Quick Start

### 1. Clone and Configure

```bash
git clone <repo-url>
cd trough

# Create production environment file
cp .env.production.example .env.production

# Edit with your values
vim .env.production
```

Required environment variables:
- `POSTGRES_PASSWORD`: Strong database password
- `PUBLIC_API_URL`: Full URL to your API (e.g., `https://your-domain.com/api`)
- `PUBLIC_GOOGLE_MAPS_API_KEY`: Google Maps API key for map view

### 2. Build and Deploy

```bash
# Build all services
docker compose -f docker-compose.prod.yml build

# Start services
docker compose -f docker-compose.prod.yml up -d

# Check status
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f
```

### 3. Run Database Migrations

```bash
# Enter the API container
docker compose -f docker-compose.prod.yml exec api sh

# Run migrations
/app/cli migrate up

# Seed initial sources
/app/cli seed

# Exit container
exit
```

### 4. Verify Deployment

```bash
# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Test API
curl http://localhost:8080/api/v1/listings
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Nginx                                │
│                    (Reverse Proxy)                           │
│                  Port 80/443 (optional)                      │
└───────────────────────┬─────────────────────────────────────┘
                        │
           ┌────────────┴────────────┐
           │                         │
           ▼                         ▼
┌──────────────────┐     ┌──────────────────────┐
│    Web (Node)    │     │      API (Go)        │
│    Port 3000     │     │     Port 8080        │
│   SvelteKit SSR  │     │  Chi Router + REST   │
└──────────────────┘     └──────────┬───────────┘
                                    │
                         ┌──────────┴───────────┐
                         │                      │
                         ▼                      ▼
              ┌──────────────────┐   ┌──────────────────┐
              │ Scraper Worker   │   │    PostgreSQL    │
              │   (Background)   │   │ + PostGIS + River│
              └──────────────────┘   └──────────────────┘
```

## Services

### API Server (`api`)
- **Port**: 8080
- **Health Check**: `GET /health`
- **Readiness Check**: `GET /ready`
- **Metrics**: `GET /metrics` (Prometheus format)

### Web Frontend (`web`)
- **Port**: 3000
- **Framework**: SvelteKit with Node adapter

### Scraper Worker (`scraper`)
- **Background service** - processes scrape jobs from River queue
- Runs scheduled scrapes daily at 2 AM UTC

### PostgreSQL (`postgres`)
- **Extensions**: PostGIS, pg_trgm
- **Data**: Persisted in `postgres_data` volume

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_USER` | Database user | `trough` |
| `POSTGRES_PASSWORD` | Database password | Required |
| `POSTGRES_DB` | Database name | `trough` |
| `API_PORT` | API server port | `8080` |
| `WEB_PORT` | Web server port | `3000` |
| `PUBLIC_API_URL` | Public API URL | Required |
| `LOG_LEVEL` | Logging level | `info` |

### With Nginx (HTTPS)

To enable the Nginx reverse proxy:

```bash
# Start with nginx profile
docker compose -f docker-compose.prod.yml --profile with-nginx up -d
```

For HTTPS, place your SSL certificates in `nginx/certs/`:
- `fullchain.pem` - Certificate chain
- `privkey.pem` - Private key

Then uncomment the HTTPS server block in `nginx/nginx.conf`.

## Monitoring

### Prometheus Metrics

The API exposes Prometheus metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

Key metrics:
- `trough_http_requests_total` - Total HTTP requests by method, path, status
- `trough_http_request_duration_seconds` - Request latency histogram
- `trough_http_active_requests` - Currently processing requests
- `trough_scrape_jobs_total` - Scrape jobs by source and status
- `trough_scrape_listings_total` - Listings scraped by source

### Health Checks

```bash
# Liveness check
curl http://localhost:8080/health

# Response includes:
# - Database connectivity
# - Memory usage
# - Goroutine count

# Readiness check
curl http://localhost:8080/ready
```

### Logs

```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Specific service
docker compose -f docker-compose.prod.yml logs -f api

# JSON logs (structured)
docker compose -f docker-compose.prod.yml logs api 2>&1 | jq .
```

## Maintenance

### Database Backup

```bash
# Create backup
docker compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U trough trough > backup.sql

# Restore backup
docker compose -f docker-compose.prod.yml exec -T postgres \
  psql -U trough trough < backup.sql
```

### Manual Scraping

```bash
# Run a scrape manually
docker compose -f docker-compose.prod.yml exec api \
  /app/cli scrape run -s bizbuysell -l 100

# Check stats
docker compose -f docker-compose.prod.yml exec api \
  /app/cli stats
```

### Updating

```bash
# Pull latest code
git pull

# Rebuild and restart
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d

# Run any new migrations
docker compose -f docker-compose.prod.yml exec api /app/cli migrate up
```

## Scaling

### Horizontal Scaling

The API server is stateless and can be scaled horizontally:

```yaml
# In docker-compose.prod.yml
api:
  deploy:
    replicas: 3
```

### Database Scaling

For high traffic, consider:
- PostgreSQL connection pooling (PgBouncer)
- Read replicas for search queries
- External managed PostgreSQL (RDS, Cloud SQL)

## Troubleshooting

### Common Issues

**Database connection errors:**
```bash
# Check postgres is running
docker compose -f docker-compose.prod.yml ps postgres

# Check connectivity
docker compose -f docker-compose.prod.yml exec api \
  sh -c 'nc -zv postgres 5432'
```

**Scraper not working:**
```bash
# Check worker logs
docker compose -f docker-compose.prod.yml logs scraper

# Check River queue
docker compose -f docker-compose.prod.yml exec postgres \
  psql -U trough -c "SELECT * FROM river_job WHERE state = 'available'"
```

**High memory usage:**
```bash
# Check container stats
docker stats

# Restart specific service
docker compose -f docker-compose.prod.yml restart api
```

## Security Recommendations

1. **Database**: Use strong passwords, restrict network access
2. **API**: Enable rate limiting, validate all inputs
3. **Nginx**: Enable HTTPS, HSTS, proper headers
4. **Docker**: Run as non-root, use security scans
5. **Secrets**: Use Docker secrets or external vault for production
