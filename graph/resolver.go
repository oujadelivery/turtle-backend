package graph

import (
	"context"
	"time"

	"turtle/internal/application/usecases"
	"turtle/internal/domain"
)

// This file will not be regenerated automatically.
// It serves as dependency injection for your app.

// Resolver is the root resolver that holds all dependencies
type Resolver struct {
	// Repositories
	userRepo         domain.UserRepository
	addressRepo      domain.AddressRepository
	otpRepo          domain.OTPRepository
	refreshTokenRepo domain.RefreshTokenRepository

	// Services
	authService *usecases.AuthenticationService

	// Cache for rate limiting
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
func NewResolver(
	userRepo domain.UserRepository,
	addressRepo domain.AddressRepository,
	otpRepo domain.OTPRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	authService *usecases.AuthenticationService,
	cache CacheService,
) *Resolver {
	return &Resolver{
		userRepo:         userRepo,
		addressRepo:      addressRepo,
		otpRepo:          otpRepo,
		refreshTokenRepo: refreshTokenRepo,
		authService:      authService,
		cache:            cache,
	}
}
