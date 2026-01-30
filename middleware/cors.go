package middleware

import (
	"net/http"
	"strings"
)

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
				// Support wildcard subdomains (e.g., *.example.com)
				if strings.HasPrefix(allowedOrigin, "*.") {
					domain := strings.TrimPrefix(allowedOrigin, "*")
					if strings.HasSuffix(origin, domain) {
						allowed = true
						break
					}
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultAllowedOrigins returns default allowed origins for development
func DefaultAllowedOrigins() []string {
	return []string{
		"http://localhost:3000",
		"http://localhost:3001",
		"http://localhost:8080",
		// Add production domains
		// "https://app.turtledelivery.com",
		// "https://www.turtledelivery.com",
	}
}

// ProductionAllowedOrigins returns allowed origins for production
func ProductionAllowedOrigins() []string {
	return []string{
		"https://app.turtledelivery.com",
		"https://www.turtledelivery.com",
		"https://*.turtledelivery.com", // Wildcard for subdomains
	}
}
