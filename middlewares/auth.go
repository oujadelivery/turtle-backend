package middlewares

import (
	"context"
	"net/http"
	"strings"

	"turtle/pkg/jwt"
)

type contextKey string

const UserContextKey contextKey = "user"

type AuthUser struct {
	UserID uint
	Role   string
	Device string
}

// AuthMiddleware extracts JWT token and adds user to context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			// No token provided - continue without auth
			next.ServeHTTP(w, r)
			return
		}

		// Extract Bearer token
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

		// Add user to context
		user := &AuthUser{
			UserID: claims.UserID,
			Role:   claims.Role,
			Device: claims.Device,
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves user from context
func GetUserFromContext(ctx context.Context) (*AuthUser, bool) {
	user, ok := ctx.Value(UserContextKey).(*AuthUser)
	return user, ok
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
