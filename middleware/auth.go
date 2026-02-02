package middleware

import (
	"context"
	"net/http"
	"strings"

	"turtle/pkg/errors"
	"turtle/pkg/jwt"
)

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No token provided - continue without auth
				next.ServeHTTP(w, r)
				return
			}

			// Extract Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, errors.ErrInvalidToken(), http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Verify token
			claims, err := jwt.VerifyToken(token)
			if err != nil {
				writeError(w, errors.ErrInvalidToken(), http.StatusUnauthorized)
				return
			}
			// clientIP := getClientIP(r)

			// Set user context
			ctx := context.WithValue(r.Context(), "userID", claims.UserID)
			ctx = context.WithValue(ctx, "role", claims.Role)
			ctx = context.WithValue(ctx, "device", claims.Device)
			// ctx = context.WithValue(r.Context(), "client_ip", claims.clientIP)

			// Continue with authenticated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth middleware that enforces authentication
func RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value("userID")
			if userID == nil || userID == "" {
				writeError(w, errors.ErrUnauthorized("Authentication required"), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole middleware that enforces role-based access
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value("role").(string)
			if !ok || role == "" {
				writeError(w, errors.ErrUnauthorized("Authentication required"), http.StatusUnauthorized)
				return
			}

			// Check if user has required role
			allowed := false
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					allowed = true
					break
				}
			}

			if !allowed {
				writeError(w, errors.ErrForbidden("Insufficient permissions"), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":"` + err.Error() + `"}`))
}
