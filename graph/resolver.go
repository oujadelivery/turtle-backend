package graph

import (
	"context"
	"time"

	"turtle/internal/application/services"
	"turtle/internal/application/usecases"
)

// This file will not be regenerated automatically.
// It serves as dependency injection for your app.

// Resolver is the root resolver that holds all dependencies
type Resolver struct {
	// ============================================================================
	// SERVICES (NEW ARCHITECTURE)
	// Services wrap repositories with DataLoader optimization
	// ============================================================================
	userService    *services.UserService
	addressService *services.AddressService
	authService    *usecases.AuthenticationService

	// ============================================================================
	// INFRASTRUCTURE
	// ============================================================================
	cache CacheService
}

// CacheService interface for caching operations
type CacheService interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error)
}

// NewResolver creates a new root resolver with all dependencies
//
// MIGRATION NOTE:
// Old signature: NewResolver(userRepo, addressRepo, otpRepo, refreshTokenRepo, authService, cache)
// New signature: NewResolver(userService, addressService, authService, cache)
//
// Repositories are now encapsulated in services!
func NewResolver(
	userService *services.UserService,
	addressService *services.AddressService,
	authService *usecases.AuthenticationService,
	cache CacheService,
) *Resolver {
	return &Resolver{
		userService:    userService,
		addressService: addressService,
		authService:    authService,
		cache:          cache,
	}
}