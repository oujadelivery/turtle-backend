package dataloader

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"turtle/internal/domain"
)

// ============================================================================
// DATALOADER MIDDLEWARE
// Adds DataLoaders to request context for per-request lifecycle
// ============================================================================

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	// Context keys for loaders
	userLoaderKey      contextKey = "userLoader"
	addressLoaderKey   contextKey = "addressLoader"
	addressByIDKey     contextKey = "addressByIDLoader"
	loaderStartTimeKey contextKey = "loaderStartTime"
)

// Loaders contains all DataLoaders for a request
type Loaders struct {
	UserLoader        *UserLoader
	AddressLoader     *AddressLoader
	AddressByIDLoader *AddressByIDLoader
}

// DataLoaderMiddleware creates DataLoaders for each request
func DataLoaderMiddleware(userRepo domain.UserRepository, addressRepo domain.AddressRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create loaders for this request
			loaders := &Loaders{
				UserLoader:        NewUserLoader(userRepo),
				AddressLoader:     NewAddressLoader(addressRepo),
				AddressByIDLoader: NewAddressByIDLoader(addressRepo),
			}

			// Add loaders to context
			ctx := context.WithValue(r.Context(), userLoaderKey, loaders.UserLoader)
			ctx = context.WithValue(ctx, addressLoaderKey, loaders.AddressLoader)
			ctx = context.WithValue(ctx, addressByIDKey, loaders.AddressByIDLoader)
			ctx = context.WithValue(ctx, loaderStartTimeKey, time.Now())

			// Continue with enhanced context
			next.ServeHTTP(w, r.WithContext(ctx))

			// Log statistics in development mode
			if isDevelopment() {
				logLoaderStats(loaders, ctx)
			}
		})
	}
}

// ============================================================================
// CONTEXT HELPERS
// Functions to retrieve loaders from context
// ============================================================================

// GetUserLoader retrieves the UserLoader from context
func GetUserLoader(ctx context.Context) (*UserLoader, error) {
	loader, ok := ctx.Value(userLoaderKey).(*UserLoader)
	if !ok || loader == nil {
		return nil, fmt.Errorf("UserLoader not found in context")
	}
	return loader, nil
}

// GetAddressLoader retrieves the AddressLoader from context
func GetAddressLoader(ctx context.Context) (*AddressLoader, error) {
	loader, ok := ctx.Value(addressLoaderKey).(*AddressLoader)
	if !ok || loader == nil {
		return nil, fmt.Errorf("AddressLoader not found in context")
	}
	return loader, nil
}

// GetAddressByIDLoader retrieves the AddressByIDLoader from context
func GetAddressByIDLoader(ctx context.Context) (*AddressByIDLoader, error) {
	loader, ok := ctx.Value(addressByIDKey).(*AddressByIDLoader)
	if !ok || loader == nil {
		return nil, fmt.Errorf("AddressByIDLoader not found in context")
	}
	return loader, nil
}

// MustGetUserLoader retrieves the UserLoader from context or panics
// Use this in resolvers where you know the loader must exist
func MustGetUserLoader(ctx context.Context) *UserLoader {
	loader, err := GetUserLoader(ctx)
	if err != nil {
		panic(err)
	}
	return loader
}

// MustGetAddressLoader retrieves the AddressLoader from context or panics
func MustGetAddressLoader(ctx context.Context) *AddressLoader {
	loader, err := GetAddressLoader(ctx)
	if err != nil {
		panic(err)
	}
	return loader
}

// MustGetAddressByIDLoader retrieves the AddressByIDLoader from context or panics
func MustGetAddressByIDLoader(ctx context.Context) *AddressByIDLoader {
	loader, err := GetAddressByIDLoader(ctx)
	if err != nil {
		panic(err)
	}
	return loader
}

// ============================================================================
// STATISTICS AND MONITORING
// ============================================================================

// logLoaderStats logs DataLoader statistics for monitoring
func logLoaderStats(loaders *Loaders, ctx context.Context) {
	startTime, ok := ctx.Value(loaderStartTimeKey).(time.Time)
	if !ok {
		return
	}

	duration := time.Since(startTime)

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📊 DataLoader Statistics")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Request Duration: %v\n", duration)
	fmt.Println("───────────────────────────────────────────────────────────")

	// User Loader stats
	userStats := loaders.UserLoader.Stats()
	fmt.Printf("👤 User Loader:\n")
	fmt.Printf("   %s\n", userStats.String())

	// Address Loader stats
	addressStats := loaders.AddressLoader.Stats()
	fmt.Printf("📍 Address Loader:\n")
	fmt.Printf("   %s\n", addressStats.String())

	// AddressByID Loader stats
	addressByIDStats := loaders.AddressByIDLoader.Stats()
	fmt.Printf("🏠 AddressByID Loader:\n")
	fmt.Printf("   %s\n", addressByIDStats.String())

	// Overall efficiency
	totalLoads := userStats.TotalLoads + addressStats.TotalLoads + addressByIDStats.TotalLoads
	totalBatches := userStats.BatchCount + addressStats.BatchCount + addressByIDStats.BatchCount

	if totalLoads > 0 && totalBatches > 0 {
		efficiency := (1 - float64(totalBatches)/float64(totalLoads)) * 100
		fmt.Println("───────────────────────────────────────────────────────────")
		fmt.Printf("💡 Overall Efficiency: %.1f%% reduction in queries\n", efficiency)
		fmt.Printf("   (%d loads → %d batches)\n", totalLoads, totalBatches)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
}

// isDevelopment checks if we're in development mode
func isDevelopment() bool {
	// You can check environment variable or config
	// For now, always return true to show stats
	return true
}

// ============================================================================
// EXAMPLE USAGE IN RESOLVERS
// ============================================================================

/*
Here's how to use DataLoaders in your resolvers:

// BEFORE (N+1 query problem):
func (r *queryResolver) MyAddresses(ctx context.Context) ([]*model.Address, error) {
	userID := getUserIDFromContext(ctx)
	addresses, err := r.Resolver.addressRepo.FindByUserID(ctx, userID)
	// ... convert and return
}

// AFTER (using DataLoader):
func (r *queryResolver) MyAddresses(ctx context.Context) ([]*model.Address, error) {
	userID := getUserIDFromContext(ctx)

	// Use DataLoader instead of repository
	loader := dataloader.MustGetAddressLoader(ctx)
	addresses, err := loader.LoadAddressesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert to GraphQL models
	result := make([]*model.Address, len(addresses))
	for i, addr := range addresses {
		result[i] = addressToGraphQL(addr)
	}

	return result, nil
}

// For loading multiple users (e.g., in a list):
func (r *queryResolver) Users(ctx context.Context, ids []string) ([]*model.User, error) {
	loader := dataloader.MustGetUserLoader(ctx)
	users, err := loader.LoadUsers(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]*model.User, len(users))
	for i, user := range users {
		result[i] = userToGraphQL(user)
	}

	return result, nil
}

BENEFITS:
✅ Batches multiple Load() calls into single database query
✅ Caches results within the request
✅ Deduplicates requests automatically
✅ Dramatically improves performance for lists
✅ No N+1 query problem

EXAMPLE PERFORMANCE IMPROVEMENT:
Query: Load 100 users with their addresses
- WITHOUT DataLoader: 1 + 100 = 101 queries
- WITH DataLoader: 1 + 1 = 2 queries
- Improvement: 98% reduction! 🚀
*/
