package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RequestLog represents a structured log entry for HTTP requests
type RequestLog struct {
	Timestamp   string `json:"timestamp"`
	RequestID   string `json:"request_id"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	RemoteAddr  string `json:"remote_addr"`
	UserAgent   string `json:"user_agent"`
	StatusCode  int    `json:"status_code"`
	Bytes       int    `json:"bytes"`
	DurationMs  int64  `json:"duration_ms"`
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// StructuredLogger is a middleware that logs requests in JSON format
func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status and bytes
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Get request ID from context
		requestID := middleware.GetReqID(r.Context())

		// Create log entry
		entry := RequestLog{
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			RequestID:   requestID,
			Method:      r.Method,
			Path:        r.URL.Path,
			RemoteAddr:  r.RemoteAddr,
			UserAgent:   r.UserAgent(),
			StatusCode:  wrapped.statusCode,
			Bytes:       wrapped.bytes,
			DurationMs:  duration.Milliseconds(),
		}

		// Log as JSON
		logBytes, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Error marshaling log entry: %v", err)
			return
		}

		log.Println(string(logBytes))
	})
}
