package main

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// loggingMiddleware logs each request with URL, status code, and duration
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a custom response writer to capture the status code
		lrw := newLoggingResponseWriter(w)

		// Record start time
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(lrw, r)

		// Calculate duration in milliseconds
		duration := time.Since(start).Milliseconds()

		// Log the request details
		log.Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Int("status", lrw.statusCode).
			Int64("duration_ms", duration).
			Msg("Request processed")
	})
}

// loggingResponseWriter is a custom ResponseWriter that captures the status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// newLoggingResponseWriter creates a new loggingResponseWriter
func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code and calls the underlying ResponseWriter's WriteHeader
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
