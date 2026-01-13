# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/cli ./cmd/cli
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/scraper ./cmd/scraper

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binaries
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/cli /app/cli
COPY --from=builder /app/bin/scraper /app/scraper
COPY --from=builder /app/migrations /app/migrations

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

CMD ["/app/api"]
