package dataloader

import (
	"context"
	"time"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
)

// ============================================================================
// USER DATALOADER
// Batches user loading to prevent N+1 queries
// ============================================================================

const (
	// DefaultWait is the default time to wait before dispatching a batch
	DefaultWait = 16 * time.Millisecond // ~60fps

	// DefaultMaxBatch is the default maximum batch size
	DefaultMaxBatch = 100
)

// UserLoader batches user loading operations
type UserLoader struct {
	*DataLoader[string, *aggregates.User]
	repo domain.UserRepository
}

// NewUserLoader creates a new UserLoader
func NewUserLoader(repo domain.UserRepository) *UserLoader {
	loader := &UserLoader{
		repo: repo,
	}

	// Create the generic DataLoader with batch function
	loader.DataLoader = NewDataLoader(
		loader.batchLoadUsers,
		DefaultWait,
		DefaultMaxBatch,
	)

	return loader
}

// batchLoadUsers is the batch function that loads multiple users at once
func (l *UserLoader) batchLoadUsers(ctx context.Context, userIDs []string) (map[string]*aggregates.User, error) {
	// Use the optimized FindByIDs method
	users, err := l.repo.FindByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// Convert to map for DataLoader
	results := make(map[string]*aggregates.User, len(users))
	for _, user := range users {
		results[user.ID()] = user
	}

	return results, nil
}

// LoadUser loads a single user by ID (with batching and caching)
func (l *UserLoader) LoadUser(ctx context.Context, userID string) (*aggregates.User, error) {
	return l.Load(ctx, userID)
}

// LoadUsers loads multiple users by IDs (with batching and caching)
func (l *UserLoader) LoadUsers(ctx context.Context, userIDs []string) ([]*aggregates.User, error) {
	return l.LoadMany(ctx, userIDs)
}

// ============================================================================
// BATCH USER REPOSITORY EXTENSION
// Add this to your UserRepository for optimal performance
// ============================================================================

/*
RECOMMENDED: Add this method to domain.UserRepository interface:

// FindByIDs finds multiple users by their IDs in a single query
FindByIDs(ctx context.Context, ids []string) ([]*aggregates.User, error)

Then implement in postgres/user_repository.go:

func (r *UserRepository) FindByIDs(ctx context.Context, ids []string) ([]*aggregates.User, error) {
	var models []UserModel

	err := r.db.WithContext(ctx).
		Preload("CaptainProfile").
		Preload("AdminProfile").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}

	// Convert to domain models
	users := make([]*aggregates.User, 0, len(models))
	for _, model := range models {
		user, err := UserToDomain(&model)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

With this optimization, the batchLoadUsers function becomes:

func (l *UserLoader) batchLoadUsers(ctx context.Context, userIDs []string) (map[string]*aggregates.User, error) {
	users, err := l.repo.FindByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	results := make(map[string]*aggregates.User, len(users))
	for _, user := range users {
		results[user.ID()] = user
	}

	return results, nil
}

This turns N queries into 1 query! 🚀
*/
