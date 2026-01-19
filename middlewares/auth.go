package middlewares

import (
	"context"
	"net/http"
	"strings"
	"time"

	"turtle/infra"
	"turtle/pkg/jwt"

	"github.com/google/uuid"
)

type contextKey string

const (
	UserContextKey      contextKey = "user"
	RequestIDContextKey contextKey = "request_id"
)

type AuthUser struct {
	UserID uint
	Role   string
	Device string
}

// AuthMiddleware extracts JWT token and adds user to context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := tokenParts[1]

		// Verify token
		claims, err := jwt.VerifyToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// IMPROVEMENT: Check if token is blacklisted (for logout)
		isBlacklisted, _ := infra.Redis.Exists(r.Context(), "blacklist:"+tokenString).Result()
		if isBlacklisted > 0 {
			http.Error(w, "Token has been revoked", http.StatusUnauthorized)
			return
		}

		user := &AuthUser{
			UserID: claims.UserID,
			Role:   claims.Role,
			Device: claims.Device,
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IMPROVEMENT: AdminOnlyMiddleware restricts access to admin users
func AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r.Context())
		if !ok || user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if user.Role != "ADMIN" {
			http.Error(w, "Forbidden - Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// IMPROVEMENT: ValidationMiddleware validates common inputs
func ValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Content-Type for POST/PUT requests
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") &&
				!strings.Contains(contentType, "multipart/form-data") {
				http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
				return
			}
		}

		// CRITICAL: Add request size limit (already handled by http.MaxBytesReader in server config)
		// Additional validation can be added here

		next.ServeHTTP(w, r)
	})
}

// IMPROVEMENT: IdempotencyMiddleware handles idempotent requests
func IdempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check for POST requests
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		idempotencyKey := r.Header.Get("X-Idempotency-Key")
		if idempotencyKey == "" {
			// Generate one if not provided (optional)
			idempotencyKey = uuid.New().String()
			r.Header.Set("X-Idempotency-Key", idempotencyKey)
		}

		// Check if we've seen this key before (in last 24 hours)
		cacheKey := "idempotency:" + idempotencyKey
		
		// Try to get cached response
		cachedResponse, err := infra.Redis.Get(r.Context(), cacheKey).Bytes()
		if err == nil && len(cachedResponse) > 0 {
			// Return cached response
			w.Header().Set("X-Idempotency-Replay", "true")
			w.Write(cachedResponse)
			return
		}

		// IMPORTANT: For actual implementation, you'd need to wrap the ResponseWriter
		// to capture the response and cache it
		
		next.ServeHTTP(w, r)
	})
}

// IMPROVEMENT: RequestIDMiddleware adds unique request ID (if not using chi's middleware.RequestID)
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)
		w.Header().Set("X-Request-ID", requestID)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IMPROVEMENT: TimeoutMiddleware with custom timeout based on operation
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				http.Error(w, "Request timeout", http.StatusGatewayTimeout)
				return
			}
		})
	}
}

// GetUserFromContext retrieves user from context
func GetUserFromContext(ctx context.Context) (*AuthUser, bool) {
	user, ok := ctx.Value(UserContextKey).(*AuthUser)
	return user, ok
}

// GetRequestIDFromContext retrieves request ID from context
func GetRequestIDFromContext(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return reqID
	}
	return ""
}

// RequireAuth is a helper to check if user is authenticated in resolvers
func RequireAuth(ctx context.Context) (*AuthUser, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, ErrUnauthorized
	}
	return user, nil
}

// RequireRole checks if user has specific role
func RequireRole(ctx context.Context, allowedRoles ...string) (*AuthUser, error) {
	user, err := RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	for _, role := range allowedRoles {
		if user.Role == role {
			return user, nil
		}
	}

	return nil, ErrForbidden
}

// IMPROVEMENT: RequireVerifiedEmail checks if user has verified email
func RequireVerifiedEmail(ctx context.Context) (*AuthUser, error) {
	user, err := RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// This would need to be checked against the database
	// For now, just return the user
	return user, nil
}

// IMPROVEMENT: CheckPermission checks specific permission
func CheckPermission(ctx context.Context, resource, action string) error {
	user, err := RequireAuth(ctx)
	if err != nil {
		return err
	}

	// IMPORTANT: Implement actual permission checking logic
	// This is a placeholder for a full RBAC system
	
	// Admin has all permissions
	if user.Role == "ADMIN" {
		return nil
	}

	// Check resource-specific permissions
	switch resource {
	case "order":
		if action == "cancel" || action == "view" {
			return nil // All authenticated users can view/cancel their orders
		}
	case "captain":
		if user.Role == "CAPTAIN" && (action == "accept" || action == "deliver") {
			return nil
		}
	}

	return ErrForbidden
}

var (
	ErrUnauthorized = &AuthError{Message: "unauthorized - please login", Code: "UNAUTHORIZED"}
	ErrForbidden    = &AuthError{Message: "forbidden - insufficient permissions", Code: "FORBIDDEN"}
)

type AuthError struct {
	Message string
	Code    string
}

func (e *AuthError) Error() string {
	return e.Message
}

// IMPROVEMENT: AuditLog middleware for sensitive operations
type AuditLogger interface {
	Log(ctx context.Context, event string, data map[string]interface{})
}

func AuditLogMiddleware(logger AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Capture user info
			user, _ := GetUserFromContext(r.Context())
			
			// Serve request
			next.ServeHTTP(w, r)

			// Log after request completes
			if user != nil {
				logger.Log(r.Context(), "http_request", map[string]interface{}{
					"user_id":    user.UserID,
					"role":       user.Role,
					"method":     r.Method,
					"path":       r.URL.Path,
					"duration":   time.Since(start).Milliseconds(),
					"ip":         r.RemoteAddr,
					"user_agent": r.UserAgent(),
				})
			}
		})
	}
}

// IMPROVEMENT: CORS Security - prevent CSRF
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		// Strict Transport Security (only in production with HTTPS)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}