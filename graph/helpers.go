package graph

import (
	"context"
	"fmt"
	"strings"
	"time"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
	pkgErrors "turtle/pkg/errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Context keys
type contextKey string

const (
	userIDContextKey      contextKey = "userID"
	userRoleContextKey    contextKey = "userRole"
	refreshTokenContextKey contextKey = "refreshToken"
)

// ============================================================================
// CONTEXT HELPERS
// ============================================================================

// getUserIDFromContext extracts user ID from context
func getUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	if !ok || userID == "" {
		return "", pkgErrors.ErrUnauthorized("user not authenticated")
	}
	return userID, nil
}

// getUserRoleFromContext extracts user role from context
func getUserRoleFromContext(ctx context.Context) (string, error) {
	role, ok := ctx.Value(userRoleContextKey).(string)
	if !ok || role == "" {
		return "", pkgErrors.ErrUnauthorized("user role not found")
	}
	return role, nil
}

// getRefreshTokenFromContext extracts refresh token from context
func getRefreshTokenFromContext(ctx context.Context) (string, error) {
	token, ok := ctx.Value(refreshTokenContextKey).(string)
	if !ok || token == "" {
		return "", pkgErrors.ErrUnauthorized("refresh token not found")
	}
	return token, nil
}

// isAdmin checks if the current user is an admin
func isAdmin(ctx context.Context) bool {
	role, err := getUserRoleFromContext(ctx)
	if err != nil {
		return false
	}
	return role == "ADMIN"
}

func determineAddressSuggestionReason(addr *aggregates.Address) string {
	if addr.IsDefault() {
		return "Your default address"
	}
	if addr.UsageCount() > 10 {
		return "Frequently used"
	}
	if addr.LastUsedAt() != nil {
		return "Recently used"
	}
	return "Suggested for you"
}

// ============================================================================
// ERROR HANDLING
// ============================================================================

// handleError converts domain errors to GraphQL errors
// handleError converts domain errors to GraphQL errors
func handleError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// Check if it's already a GraphQL error
	if _, ok := err.(*gqlerror.Error); ok {
		return err
	}

	// Handle domain errors with proper codes
	if appErr, ok := pkgErrors.GetAppError(err); ok {
		return &gqlerror.Error{
			Message: appErr.Message,
			Extensions: map[string]interface{}{
				"code":    string(appErr.Code),
				"details": appErr.Details,
			},
			Path: graphql.GetPath(ctx),
		}
	}

	// Handle common domain errors
	switch err {
	case domain.ErrNotFound:
		return &gqlerror.Error{
			Message: "Resource not found",
			Extensions: map[string]interface{}{
				"code": "NOT_FOUND",
			},
			Path: graphql.GetPath(ctx),
		}
	case domain.ErrAlreadyExists:
		return &gqlerror.Error{
			Message: "Resource already exists",
			Extensions: map[string]interface{}{
				"code": "ALREADY_EXISTS",
			},
			Path: graphql.GetPath(ctx),
		}
	case domain.ErrConcurrentModification:
		return &gqlerror.Error{
			Message: "Resource was modified by another request. Please try again.",
			Extensions: map[string]interface{}{
				"code": "CONCURRENT_MODIFICATION",
			},
			Path: graphql.GetPath(ctx),
		}
	}

	// ⭐ FIX: Return actual error message instead of generic message
	errorMessage := err.Error()
	errorCode := "INTERNAL_ERROR"
	errMsg := strings.ToLower(errorMessage)
	
	// Classify error by message content
	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "too many") {
		errorCode = "RATE_LIMIT_EXCEEDED"
	} else if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "not authenticated") {
		errorCode = "UNAUTHENTICATED"
	} else if strings.Contains(errMsg, "required") || strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "must") {
		errorCode = "BAD_REQUEST"
	} else if strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "permission") {
		errorCode = "FORBIDDEN"
	} else if strings.Contains(errMsg, "not found") {
		errorCode = "NOT_FOUND"
	} else if strings.Contains(errMsg, "already exists") || strings.Contains(errMsg, "duplicate") {
		errorCode = "ALREADY_EXISTS"
	}

	// Log for debugging
	fmt.Printf("GraphQL Error [%s]: %v\n", errorCode, err)

	// ⭐ Return actual error message to client
	return &gqlerror.Error{
		Message: errorMessage,  // ⭐ Changed from "An internal error occurred"
		Extensions: map[string]interface{}{
			"code": errorCode,
		},
		Path: graphql.GetPath(ctx),
	}
}

const (
	// Version of the API
	APIVersion = "v1.0.0"
)
func distance(lat1, lng1, lat2, lng2 float64) float64 {
	// Convert degrees to approximate kilometers
	// At equator: 1 degree latitude ≈ 111 km, 1 degree longitude ≈ 111 km
	latDiff := (lat2 - lat1) * 111.0
	lngDiff := (lng2 - lng1) * 111.0

	// Euclidean distance
	return (latDiff*latDiff + lngDiff*lngDiff)
}


func floatPtr(f float64) *float64 {
	return &f
}

func getRoleFromContext(ctx context.Context) string {
	if role, ok := ctx.Value("role").(string); ok {
		return role
	}
	return ""
}

// ============================================================================
// POINTER HELPERS
// ============================================================================

// stringValue safely gets string value from pointer
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// stringPtr creates a pointer to string
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// intValue safely gets int value from pointer
func intValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// intPtr creates a pointer to int
func intPtr(i int) *int {
	return &i
}

// boolValue safely gets bool value from pointer
func boolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// boolPtr creates a pointer to bool
func boolPtr(b bool) *bool {
	return &b
}

// float64Value safely gets float64 value from pointer
func float64Value(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// float64Ptr creates a pointer to float64
func float64Ptr(f float64) *float64 {
	return &f
}

// timeValue safely converts time pointer
func timeValue(t *int64) *int64 {
	return t
}

func TimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}