package middleware

import (
	"fmt"
	"net/http"
	"time"

	"turtle/internal/infrastructure/cache"
)

// RateLimitMiddleware implements rate limiting per user/IP
func RateLimitMiddleware(cache cache.CacheService, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get identifier (userID or IP)
			identifier := getIdentifier(r)

			// Check rate limit
			allowed, remaining, err := cache.CheckRateLimit(r.Context(), fmt.Sprintf("ratelimit:%s", identifier), limit, window)
			if err != nil {
				// Log error but don't block request
				fmt.Printf("Rate limit check failed: %v\n", err)
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(window).Unix()))

			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				writeError(w, fmt.Errorf("rate limit exceeded"), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getIdentifier returns userID if authenticated, otherwise IP address
func getIdentifier(r *http.Request) string {
	// Try to get userID from context first
	if userID, ok := r.Context().Value("userID").(string); ok && userID != "" {
		return fmt.Sprintf("user:%s", userID)
	}

	// Fall back to IP address
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}
	return fmt.Sprintf("ip:%s", ip)
}

// OperationRateLimiter limits specific GraphQL operations
func OperationRateLimiter(cache cache.CacheService, operationLimits map[string]RateLimit) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract operation name from request
			operationName := r.URL.Query().Get("operationName")
			if operationName == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if operation has specific rate limit
			rateLimit, exists := operationLimits[operationName]
			if !exists {
				next.ServeHTTP(w, r)
				return
			}

			// Get identifier
			identifier := getIdentifier(r)
			key := fmt.Sprintf("op:%s:%s", operationName, identifier)

			// Check rate limit
			allowed, remaining, err := cache.CheckRateLimit(r.Context(), key, rateLimit.Limit, rateLimit.Window)
			if err != nil {
				fmt.Printf("Operation rate limit check failed: %v\n", err)
				next.ServeHTTP(w, r)
				return
			}

			// Set headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimit.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rateLimit.Window.Seconds())))
				writeError(w, fmt.Errorf("operation rate limit exceeded"), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit represents rate limit configuration
type RateLimit struct {
	Limit  int
	Window time.Duration
}

// DefaultOperationLimits returns default rate limits for operations
func DefaultOperationLimits() map[string]RateLimit {
	return map[string]RateLimit{
		"requestOTP": {
			Limit:  3,
			Window: time.Hour,
		},
		"verifyOTP": {
			Limit:  5,
			Window: 15 * time.Minute,
		},
		"socialLogin": {
			Limit:  10,
			Window: time.Hour,
		},
		"createAddress": {
			Limit:  20,
			Window: time.Hour,
		},
	}
}
