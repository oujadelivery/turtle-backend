package services

import (
	"context"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
	"turtle/internal/infrastructure/dataloader"
	pkgErrors "turtle/pkg/errors"
)

// ============================================================================
// ADDRESS SERVICE
// Service layer that handles address operations with DataLoader optimization
// ============================================================================

// AddressService provides address operations with built-in DataLoader support
type AddressService struct {
	repo domain.AddressRepository
}

// NewAddressService creates a new AddressService
func NewAddressService(repo domain.AddressRepository) *AddressService {
	return &AddressService{
		repo: repo,
	}
}

// ============================================================================
// QUERY OPERATIONS
// ============================================================================

// GetAddress fetches a single address by ID using DataLoader
func (s *AddressService) GetAddress(ctx context.Context, addressID string) (*aggregates.Address, error) {
	loader := dataloader.MustGetAddressByIDLoader(ctx)
	return loader.LoadAddress(ctx, addressID)
}

// GetAddresses fetches multiple addresses by IDs using DataLoader
func (s *AddressService) GetAddresses(ctx context.Context, addressIDs []string) ([]*aggregates.Address, error) {
	loader := dataloader.MustGetAddressByIDLoader(ctx)

	// Load all addresses using LoadMany
	return loader.LoadMany(ctx, addressIDs)
}

// GetUserAddresses fetches all addresses for a user using DataLoader
// 🚀 CRITICAL: This prevents N+1 queries when loading addresses for multiple users
func (s *AddressService) GetUserAddresses(ctx context.Context, userID string) ([]*aggregates.Address, error) {
	// Use AddressLoader which batches by userID
	loader := dataloader.MustGetAddressLoader(ctx)
	addresses, err := loader.LoadAddressesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Prime the AddressByID loader for individual lookups
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	for _, address := range addresses {
		addressByIDLoader.Prime(address.ID(), address)
	}

	return addresses, nil
}

// GetDefaultAddress fetches user's default address
func (s *AddressService) GetDefaultAddress(ctx context.Context, userID string) (*aggregates.Address, error) {
	// Default address lookup uses repository (single query with WHERE is_default = true)
	address, err := s.repo.FindDefaultByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Prime the cache with this address
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Prime(address.ID(), address)

	return address, nil
}

// GetSuggestedAddresses returns smart address suggestions based on usage patterns
func (s *AddressService) GetSuggestedAddresses(ctx context.Context, userID string, limit int) ([]*aggregates.Address, error) {
	// Smart suggestions use repository (complex algorithm)
	addresses, err := s.repo.FindSuggestedAddresses(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	// Prime both caches with suggested addresses
	// addressLoader := dataloader.MustGetAddressLoader(ctx)
	dataloader.MustGetAddressLoader(ctx)
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)

	for _, address := range addresses {
		addressByIDLoader.Prime(address.ID(), address)
	}

	return addresses, nil
}

// FindNearestAddresses finds nearest addresses to a given location
func (s *AddressService) FindNearestAddresses(ctx context.Context, userID string, lat, lng float64, limit int) ([]*aggregates.Address, error) {
	// Geospatial query uses repository
	addresses, err := s.repo.FindNearest(ctx, userID, lat, lng, limit)
	if err != nil {
		return nil, err
	}

	// Prime cache with nearest addresses
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	for _, address := range addresses {
		addressByIDLoader.Prime(address.ID(), address)
	}

	return addresses, nil
}

// SearchAddresses searches addresses with pagination
func (s *AddressService) SearchAddresses(ctx context.Context, userID, query string, limit, offset int) ([]*aggregates.Address, int64, error) {
	// Full-text search uses repository
	addresses, total, err := s.repo.SearchAddresses(ctx, userID, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Prime cache with search results
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	for _, address := range addresses {
		addressByIDLoader.Prime(address.ID(), address)
	}

	return addresses, total, nil
}

// GetAddressStats returns address statistics for a user
func (s *AddressService) GetAddressStats(ctx context.Context, userID string) (*domain.AddressStats, error) {
	stats, err := s.repo.GetAddressStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Prime cache with stats addresses
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	if stats.MostUsedAddress != nil {
		addressByIDLoader.Prime(stats.MostUsedAddress.ID(), stats.MostUsedAddress)
	}
	if stats.DefaultAddress != nil {
		addressByIDLoader.Prime(stats.DefaultAddress.ID(), stats.DefaultAddress)
	}
	for _, address := range stats.RecentAddresses {
		addressByIDLoader.Prime(address.ID(), address)
	}

	return stats, nil
}

// ============================================================================
// MUTATION OPERATIONS (with cache invalidation)
// ============================================================================

// CreateAddress creates a new address
func (s *AddressService) CreateAddress(ctx context.Context, userID string, address *aggregates.Address) error {
	// Verify ownership
	if address.UserID() != userID {
		return pkgErrors.ErrForbidden("Cannot create address for another user")
	}

	// Create address
	err := s.repo.Create(ctx, address)
	if err != nil {
		return err
	}

	// Clear user's address list cache
	addressLoader := dataloader.MustGetAddressLoader(ctx)
	addressLoader.Clear(userID)

	// Prime the AddressByID cache with new address
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Prime(address.ID(), address)

	return nil
}

// UpdateAddress updates an existing address
func (s *AddressService) UpdateAddress(ctx context.Context, userID string, address *aggregates.Address) (*aggregates.Address, error) {
	// Verify ownership
	if address.UserID() != userID {
		return nil, pkgErrors.ErrForbidden("You don't have permission to modify this address")
	}

	// Update address
	err := s.repo.Update(ctx, address)
	if err != nil {
		return nil, err
	}

	// Clear both caches
	addressLoader := dataloader.MustGetAddressLoader(ctx)
	addressLoader.Clear(userID)

	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Clear(address.ID())

	// Reload updated address
	return addressByIDLoader.LoadAddress(ctx, address.ID())
}

// DeleteAddress soft deletes an address
func (s *AddressService) DeleteAddress(ctx context.Context, userID, addressID string) error {
	// Fetch address to verify ownership
	address, err := s.GetAddress(ctx, addressID)
	if err != nil {
		return err
	}

	// Verify ownership
	if address.UserID() != userID {
		return pkgErrors.ErrForbidden("You don't have permission to delete this address")
	}

	// Delete address
	err = s.repo.Delete(ctx, addressID)
	if err != nil {
		return err
	}

	// Clear both caches
	addressLoader := dataloader.MustGetAddressLoader(ctx)
	addressLoader.Clear(userID)

	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Clear(addressID)

	return nil
}

// SetDefaultAddress sets an address as default
func (s *AddressService) SetDefaultAddress(ctx context.Context, userID, addressID string) (*aggregates.Address, error) {
	// Fetch address
	address, err := s.GetAddress(ctx, addressID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if address.UserID() != userID {
		return nil, pkgErrors.ErrForbidden("You don't have permission to modify this address")
	}

	// Unset all defaults first
	err = s.repo.UnsetDefault(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Set as default
	address.SetAsDefault()
	err = s.repo.Update(ctx, address)
	if err != nil {
		return nil, err
	}

	// Clear caches (all user's addresses affected)
	addressLoader := dataloader.MustGetAddressLoader(ctx)
	addressLoader.Clear(userID)

	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Clear(addressID)

	// Reload updated address
	return addressByIDLoader.LoadAddress(ctx, addressID)
}

// VerifyAddress marks an address location as verified
func (s *AddressService) VerifyAddress(ctx context.Context, userID, addressID string) (*aggregates.Address, error) {
	// Fetch address
	address, err := s.GetAddress(ctx, addressID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if address.UserID() != userID {
		return nil, pkgErrors.ErrForbidden("You don't have permission to verify this address")
	}

	// Mark as verified
	address.MarkAsVerified()

	// Save changes
	err = s.repo.Update(ctx, address)
	if err != nil {
		return nil, err
	}

	// Clear caches
	addressLoader := dataloader.MustGetAddressLoader(ctx)
	addressLoader.Clear(userID)

	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Clear(addressID)

	// Reload updated address
	return addressByIDLoader.LoadAddress(ctx, addressID)
}

// IncrementUsage increments address usage statistics
func (s *AddressService) IncrementUsage(ctx context.Context, addressID string) error {
	// Increment usage (lightweight atomic operation)
	err := s.repo.IncrementUsage(ctx, addressID)
	if err != nil {
		return err
	}

	// Clear cache (stats changed)
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Clear(addressID)

	return nil
}

// ============================================================================
// BATCH OPERATIONS
// ============================================================================

// CreateAddressBatch creates multiple addresses at once
func (s *AddressService) CreateAddressBatch(ctx context.Context, userID string, addresses []*aggregates.Address) error {
	// Verify all addresses belong to the user
	for _, address := range addresses {
		if address.UserID() != userID {
			return pkgErrors.ErrForbidden("Cannot create addresses for another user")
		}
	}

	// Create all addresses
	for _, address := range addresses {
		err := s.repo.Create(ctx, address)
		if err != nil {
			return err
		}

		// Prime cache for each new address
		addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
		addressByIDLoader.Prime(address.ID(), address)
	}

	// Clear user's address list cache
	addressLoader := dataloader.MustGetAddressLoader(ctx)
	addressLoader.Clear(userID)

	return nil
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// InvalidateUserAddressCache clears all address caches for a user
func (s *AddressService) InvalidateUserAddressCache(ctx context.Context, userID string) {
	addressLoader := dataloader.MustGetAddressLoader(ctx)
	addressLoader.Clear(userID)
}

// InvalidateAddressCache clears specific address from cache
func (s *AddressService) InvalidateAddressCache(ctx context.Context, addressID string) {
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Clear(addressID)
}

// PrimeCache adds an address to the cache
func (s *AddressService) PrimeCache(ctx context.Context, address *aggregates.Address) {
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)
	addressByIDLoader.Prime(address.ID(), address)
}

// GetLoaderStats returns DataLoader statistics for monitoring
func (s *AddressService) GetLoaderStats(ctx context.Context) (addressLoaderStats, addressByIDLoaderStats dataloader.LoaderStats) {
	addressLoader := dataloader.MustGetAddressLoader(ctx)
	addressByIDLoader := dataloader.MustGetAddressByIDLoader(ctx)

	return addressLoader.Stats(), addressByIDLoader.Stats()
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// ValidateAddress validates address data before operations
func (s *AddressService) ValidateAddress(address *aggregates.Address) error {
	// Add custom validation logic here
	if address.City() == "" {
		return pkgErrors.ErrInvalidInput("city", "City is required")
	}
	if address.State() == "" {
		return pkgErrors.ErrInvalidInput("state", "State is required")
	}
	return nil
}

// CalculateConfidence calculates confidence score for address suggestions
func (s *AddressService) CalculateConfidence(address *aggregates.Address) float64 {
	// Simple confidence score based on usage count
	usageCount := float64(address.UsageCount())

	if usageCount == 0 {
		return 0.5 // 50% confidence for never used
	}

	// Cap at 100% confidence
	confidence := 0.5 + (usageCount / 100.0)
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GenerateSuggestionReason generates human-readable reason for suggestion
func (s *AddressService) GenerateSuggestionReason(address *aggregates.Address) string {
	if address.IsDefault() {
		return "Your default address"
	}

	if address.UsageCount() > 10 {
		return "Frequently used"
	}

	if address.UsageCount() > 0 {
		return "Previously used"
	}

	return "Available address"
}
