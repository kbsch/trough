package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// httpRequestsTotal counts all HTTP requests
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trough_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// httpRequestDuration tracks request latency
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trough_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// httpRequestSize tracks request body sizes
	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trough_http_request_size_bytes",
			Help:    "HTTP request body size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// httpResponseSize tracks response body sizes
	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trough_http_response_size_bytes",
			Help:    "HTTP response body size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// activeRequests tracks currently processing requests
	activeRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trough_http_active_requests",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// Scraper metrics
	ScrapeJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trough_scrape_jobs_total",
			Help: "Total number of scrape jobs by source and status",
		},
		[]string{"source", "status"},
	)

	ScrapeListingsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trough_scrape_listings_total",
			Help: "Total number of listings scraped by source",
		},
		[]string{"source", "type"},
	)

	ScrapeDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trough_scrape_duration_seconds",
			Help:    "Duration of scrape jobs in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800},
		},
		[]string{"source"},
	)

	// Database metrics
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trough_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"query_type"},
	)

	// Listing metrics
	ListingsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trough_listings_total",
			Help: "Total number of listings by source",
		},
		[]string{"source", "active"},
	)
)

// metricsResponseWriter wraps http.ResponseWriter to capture metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// Metrics is a middleware that collects Prometheus metrics
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track active requests
		activeRequests.Inc()
		defer activeRequests.Dec()

		// Wrap response writer
		wrapped := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get route pattern for consistent labeling
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = r.URL.Path
		}

		// Record metrics
		httpRequestsTotal.WithLabelValues(
			r.Method,
			routePattern,
			strconv.Itoa(wrapped.statusCode),
		).Inc()

		httpRequestDuration.WithLabelValues(
			r.Method,
			routePattern,
		).Observe(duration)

		httpResponseSize.WithLabelValues(
			r.Method,
			routePattern,
		).Observe(float64(wrapped.bytes))

		if r.ContentLength > 0 {
			httpRequestSize.WithLabelValues(
				r.Method,
				routePattern,
			).Observe(float64(r.ContentLength))
		}
	})
}
