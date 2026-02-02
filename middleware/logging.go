package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Log request
			fmt.Printf("[%s] %s %s - Started\n",
				start.Format(time.RFC3339),
				r.Method,
				r.URL.Path,
			)

			// If this is a GraphQL request, log the query
			if r.Method == "POST" && r.URL.Path == "/graphql" {
				body, err := io.ReadAll(r.Body)
				if err == nil {
					// Restore body for actual handler
					r.Body = io.NopCloser(bytes.NewBuffer(body))

					// Log query (truncate if too long)
					query := string(body)
					if len(query) > 500 {
						query = query[:500] + "..."
					}
					fmt.Printf("  Query: %s\n", query)
				}
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log response
			duration := time.Since(start)
			fmt.Printf("[%s] %s %s - %d %s (%.2fms)\n",
				time.Now().Format(time.RFC3339),
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				http.StatusText(wrapped.statusCode),
				float64(duration.Microseconds())/1000,
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate request ID (use UUID in production)
			requestID := fmt.Sprintf("%d", time.Now().UnixNano())

			// Set in context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "requestID", requestID)

			// Set in response header
			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// StructuredLogger provides structured logging
type StructuredLogger struct {
	RequestID  string
	UserID     string
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
	Error      error
}

// Log outputs structured log
func (l *StructuredLogger) Log() {
	status := "success"
	if l.Error != nil {
		status = "error"
	}

	fmt.Printf(`{"request_id":"%s","user_id":"%s","method":"%s","path":"%s","status":%d,"duration_ms":%.2f,"status_text":"%s"}\n`,
		l.RequestID,
		l.UserID,
		l.Method,
		l.Path,
		l.StatusCode,
		float64(l.Duration.Microseconds())/1000,
		status,
	)

	if l.Error != nil {
		fmt.Printf(`{"request_id":"%s","error":"%s"}\n`, l.RequestID, l.Error.Error())
	}
}
