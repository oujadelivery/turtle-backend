package dataloader

import (
	"context"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
)

// ============================================================================
// ADDRESS DATALOADER
// Batches address loading to prevent N+1 queries
// ============================================================================

// AddressLoader batches address loading operations
// Note: This loader batches by userID, not addressID
type AddressLoader struct {
	*DataLoader[string, []*aggregates.Address]
	repo domain.AddressRepository
}

// NewAddressLoader creates a new AddressLoader
func NewAddressLoader(repo domain.AddressRepository) *AddressLoader {
	loader := &AddressLoader{
		repo: repo,
	}

	// Create the generic DataLoader with batch function
	loader.DataLoader = NewDataLoader(
		loader.batchLoadAddresses,
		DefaultWait,
		DefaultMaxBatch,
	)

	return loader
}

// batchLoadAddresses is the batch function that loads addresses for multiple users at once
func (l *AddressLoader) batchLoadAddresses(ctx context.Context, userIDs []string) (map[string][]*aggregates.Address, error) {
	// Use the optimized FindByUserIDs method
	return l.repo.FindByUserIDs(ctx, userIDs)
}

// LoadAddressesByUserID loads all addresses for a user (with batching and caching)
func (l *AddressLoader) LoadAddressesByUserID(ctx context.Context, userID string) ([]*aggregates.Address, error) {
	return l.Load(ctx, userID)
}

// ============================================================================
// ADDRESS BY ID LOADER
// For loading individual addresses by their ID
// ============================================================================

// AddressByIDLoader batches individual address loading operations
type AddressByIDLoader struct {
	*DataLoader[string, *aggregates.Address]
	repo domain.AddressRepository
}

// NewAddressByIDLoader creates a new AddressByIDLoader
func NewAddressByIDLoader(repo domain.AddressRepository) *AddressByIDLoader {
	loader := &AddressByIDLoader{
		repo: repo,
	}

	// Create the generic DataLoader with batch function
	loader.DataLoader = NewDataLoader(
		loader.batchLoadAddressesByID,
		DefaultWait,
		DefaultMaxBatch,
	)

	return loader
}

// batchLoadAddressesByID loads multiple addresses by their IDs
func (l *AddressByIDLoader) batchLoadAddressesByID(ctx context.Context, addressIDs []string) (map[string]*aggregates.Address, error) {
	// Use the optimized FindByIDs method
	addresses, err := l.repo.FindByIDs(ctx, addressIDs)
	if err != nil {
		return nil, err
	}

	// Convert to map for DataLoader
	results := make(map[string]*aggregates.Address, len(addresses))
	for _, address := range addresses {
		results[address.ID()] = address
	}

	return results, nil
}

// LoadAddress loads a single address by ID (with batching and caching)
func (l *AddressByIDLoader) LoadAddress(ctx context.Context, addressID string) (*aggregates.Address, error) {
	return l.Load(ctx, addressID)
}

// ============================================================================
// BATCH ADDRESS REPOSITORY EXTENSION
// Add these methods to AddressRepository for optimal performance
// ============================================================================

/*
RECOMMENDED: Add these methods to domain.AddressRepository interface:

// FindByUserIDs finds addresses for multiple users in a single query
FindByUserIDs(ctx context.Context, userIDs []string) (map[string][]*aggregates.Address, error)

// FindByIDs finds multiple addresses by their IDs in a single query
FindByIDs(ctx context.Context, ids []string) ([]*aggregates.Address, error)

Then implement in postgres/address_repository.go:

func (r *AddressRepository) FindByUserIDs(ctx context.Context, userIDs []string) (map[string][]*aggregates.Address, error) {
	var models []AddressModel

	err := r.db.WithContext(ctx).
		Where("user_id IN ? AND deleted_at IS NULL", userIDs).
		Order("user_id, is_default DESC, last_used_at DESC NULLS LAST").
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find addresses: %w", err)
	}

	// Group by user ID
	results := make(map[string][]*aggregates.Address)
	for _, model := range models {
		address, err := AddressToDomain(&model)
		if err != nil {
			continue
		}
		results[model.UserID] = append(results[model.UserID], address)
	}

	return results, nil
}

func (r *AddressRepository) FindByIDs(ctx context.Context, ids []string) ([]*aggregates.Address, error) {
	var models []AddressModel

	err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find addresses: %w", err)
	}

	addresses := make([]*aggregates.Address, 0, len(models))
	for _, model := range models {
		address, err := AddressToDomain(&model)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

With these optimizations:
- Loading addresses for 100 users: 100 queries → 1 query
- Loading 50 individual addresses: 50 queries → 1 query

Performance improvement: ~99% reduction in database queries! 🚀
*/
