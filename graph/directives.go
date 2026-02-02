package graph

import (
	"context"
	"fmt"
	"time"

	pkgErrors "turtle/pkg/errors"

	"github.com/99designs/gqlgen/graphql"
)

// ============================================================================
// DIRECTIVE IMPLEMENTATIONS
// ============================================================================

// Auth directive - works with your existing middleware
func (r *Resolver) Auth(ctx context.Context, obj interface{}, next graphql.Resolver, role *string) (interface{}, error) {
	// Check authentication (middleware already set user_id)
	userID := ctx.Value("user_id")
	if userID == nil || userID == "" {
		return nil, pkgErrors.ErrUnauthorized("Authentication required")
	}

	// Check role if specified
	if role != nil && *role != "" {
		userRole := ctx.Value("user_role")
		if userRole == nil || userRole.(string) != *role {
			return nil, pkgErrors.ErrForbidden("Requires role: " + *role)
		}
	}

	return next(ctx)
}

// RateLimit directive - applies rate limiting
func (r *Resolver) RateLimit(ctx context.Context, obj interface{}, next graphql.Resolver, limit int, window int) (interface{}, error) {
	// Get identifier for rate limiting
	identifier := r.getRateLimitIdentifier(ctx)

	// Generate rate limit key
	fieldName := graphql.GetFieldContext(ctx).Field.Name
	rateLimitKey := fmt.Sprintf("ratelimit:%s:%s", fieldName, identifier)

	// Check rate limit using cache
	allowed, remaining, err := r.cache.CheckRateLimit(
		ctx,
		rateLimitKey,
		limit,
		time.Duration(window)*time.Second,
	)

	if err != nil {
		// Log error but allow request (fail open for availability)
		fmt.Printf("⚠️  Rate limit check error: %v\n", err)
		return next(ctx)
	}

	if !allowed {
		return nil, pkgErrors.ErrRateLimitExceeded(fieldName, window)
	}

	// Add rate limit info to response headers
	operationCtx := graphql.GetOperationContext(ctx)
	if operationCtx != nil && operationCtx.Headers != nil {
		operationCtx.Headers.Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		operationCtx.Headers.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		operationCtx.Headers.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Duration(window)*time.Second).Unix()))
	}

	return next(ctx)
}

// HasPermission directive - checks specific permissions
func (r *Resolver) HasPermission(ctx context.Context, obj interface{}, next graphql.Resolver, permissions []string) (interface{}, error) {
	// Get user permissions from context
	userPermissions, ok := ctx.Value("user_permissions").([]string)
	if !ok || len(userPermissions) == 0 {
		return nil, pkgErrors.ErrForbidden("Insufficient permissions")
	}

	// Check if user has required permissions
	hasPermission := false
	for _, requiredPerm := range permissions {
		for _, userPerm := range userPermissions {
			if userPerm == requiredPerm {
				hasPermission = true
				break
			}
		}
		if hasPermission {
			break
		}
	}

	if !hasPermission {
		return nil, pkgErrors.ErrForbidden(fmt.Sprintf("Required permissions: %v", permissions))
	}

	return next(ctx)
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// getRateLimitIdentifier gets identifier for rate limiting
// Uses user ID if authenticated, otherwise IP address
func (r *Resolver) getRateLimitIdentifier(ctx context.Context) string {
	// Try user ID first (authenticated requests)
	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok && uid != "" {
			return fmt.Sprintf("user:%s", uid)
		}
	}

	// Fall back to IP address (unauthenticated)
	if ip := ctx.Value("client_ip"); ip != nil {
		if ipStr, ok := ip.(string); ok && ipStr != "" {
			return fmt.Sprintf("ip:%s", ipStr)
		}
	}

	// Default
	return "unknown"
}
