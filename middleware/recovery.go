package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"
)

// RecoveryMiddleware recovers from panics and returns 500 error
func RecoveryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log panic with stack trace
					fmt.Printf("PANIC: %v\n", err)
					fmt.Printf("Stack trace:\n%s\n", debug.Stack())

					// Get request ID if available
					requestID := ""
					if reqID, ok := r.Context().Value("requestID").(string); ok {
						requestID = reqID
					}

					// Log structured error
					fmt.Printf(`{"request_id":"%s","error":"panic","message":"%v"}\n`, requestID, err)

					// Return 500 error to client
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{
						"errors": [{
							"message": "Internal server error",
							"extensions": {
								"code": "INTERNAL_SERVER_ERROR"
							}
						}]
					}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds timeout to requests
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create channel to signal completion
			done := make(chan bool)

			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				done <- true
			}()

			select {
			case <-done:
				// Request completed successfully
				return
			case <-ctx.Done():
				// Request timed out
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusGatewayTimeout)
				w.Write([]byte(`{
					"errors": [{
						"message": "Request timeout",
						"extensions": {
							"code": "TIMEOUT"
						}
					}]
				}`))
			}
		})
	}
}
