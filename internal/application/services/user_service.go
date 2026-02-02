package services

import (
	"context"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
	"turtle/internal/domain/valueobjects"
	"turtle/internal/infrastructure/dataloader"
	pkgErrors "turtle/pkg/errors"
)

// ============================================================================
// USER SERVICE
// Service layer that handles user operations with DataLoader optimization
// ============================================================================

// UserService provides user operations with built-in DataLoader support
type UserService struct {
	repo domain.UserRepository
}

// NewUserService creates a new UserService
func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

// ============================================================================
// QUERY OPERATIONS
// ============================================================================

// HealthCheck verifies the service and underlying repository are operational
func (s *UserService) HealthCheck(ctx context.Context) error {
	return s.repo.HealthCheck(ctx)
}

// GetUser fetches a single user by ID using DataLoader
func (s *UserService) GetUser(ctx context.Context, userID string) (*aggregates.User, error) {
	loader := dataloader.MustGetUserLoader(ctx)
	return loader.LoadUser(ctx, userID)
}

// GetUsers fetches multiple users by IDs using DataLoader
func (s *UserService) GetUsers(ctx context.Context, userIDs []string) ([]*aggregates.User, error) {
	loader := dataloader.MustGetUserLoader(ctx)
	return loader.LoadUsers(ctx, userIDs)
}

// GetUserByEmail fetches a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*aggregates.User, error) {
	// Email lookup doesn't benefit from DataLoader (unique lookup)
	return s.repo.FindByEmail(ctx, email)
}

// GetUserByPhone fetches a user by phone
func (s *UserService) GetUserByPhone(ctx context.Context, phone string) (*aggregates.User, error) {
	// Phone lookup doesn't benefit from DataLoader (unique lookup)
	return s.repo.FindByPhone(ctx, phone)
}

// SearchUsers searches users with pagination and cache priming
func (s *UserService) SearchUsers(ctx context.Context, query, role string, limit, offset int) ([]*aggregates.User, int64, error) {
	// Complex search uses repository directly
	users, total, err := s.repo.Search(ctx, query, role, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Prime the DataLoader cache with search results
	loader := dataloader.MustGetUserLoader(ctx)
	for _, user := range users {
		loader.Prime(user.ID(), user)
	}

	return users, total, nil
}

// FindCaptainsNearby finds available captains within radius
func (s *UserService) FindCaptainsNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*aggregates.User, error) {
	// Geospatial query uses repository directly
	captains, err := s.repo.FindCaptainsNearby(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, err
	}

	// Prime the cache with nearby captains
	loader := dataloader.MustGetUserLoader(ctx)
	for _, captain := range captains {
		loader.Prime(captain.ID(), captain)
	}

	return captains, nil
}

// FindCaptainsByStatus finds captains by KYC status
func (s *UserService) FindCaptainsByStatus(ctx context.Context, status string) ([]*aggregates.User, error) {
	// Status-based query uses repository
	captains, err := s.repo.FindCaptainsByStatus(ctx, status)
	if err != nil {
		return nil, err
	}

	// Prime the cache
	loader := dataloader.MustGetUserLoader(ctx)
	for _, captain := range captains {
		loader.Prime(captain.ID(), captain)
	}

	return captains, nil
}

// GetUserStats returns aggregated user statistics
func (s *UserService) GetUserStats(ctx context.Context) (*domain.UserStats, error) {
	return s.repo.GetUserStats(ctx)
}

// ============================================================================
// MUTATION OPERATIONS (with cache invalidation)
// ============================================================================

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, user *aggregates.User) error {
	err := s.repo.Create(ctx, user)
	if err != nil {
		return err
	}

	// Prime the cache with new user
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Prime(user.ID(), user)

	return nil
}

// UpdateProfile updates user profile and invalidates cache
func (s *UserService) UpdateProfile(ctx context.Context, userID, firstName, lastName, profilePic string) (*aggregates.User, error) {
	// Load user
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update profile
	err = user.UpdateProfile(firstName, lastName, profilePic)
	if err != nil {
		return nil, err
	}

	// Save changes
	err = s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate cache and reload
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return loader.LoadUser(ctx, userID)
}

// UpdateVehicle updates captain's vehicle information
func (s *UserService) UpdateVehicle(ctx context.Context, userID string, vehicleType aggregates.VehicleType, vehicleNumber, vehicleModel string) (*aggregates.User, error) {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.IsCaptain() {
		return nil, pkgErrors.ErrForbidden("Only captains can update vehicle information")
	}

	err = user.UpdateVehicleInfo(vehicleType, vehicleNumber, vehicleModel)
	if err != nil {
		return nil, err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate and reload
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return loader.LoadUser(ctx, userID)
}

// SubmitKYC submits KYC documents for captain
func (s *UserService) SubmitKYC(ctx context.Context, userID string, documents map[string]string) (*aggregates.User, error) {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.IsCaptain() {
		return nil, pkgErrors.ErrForbidden("Only captains can submit KYC")
	}

	err = user.SubmitKYCDocuments(documents)
	if err != nil {
		return nil, err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate and reload
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return loader.LoadUser(ctx, userID)
}

// GoOnline sets captain as available
func (s *UserService) GoOnline(ctx context.Context, userID string, location *valueobjects.Location) error {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	err = user.GoOnline(location)
	if err != nil {
		return err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return err
	}

	// Invalidate cache (status changed)
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return nil
}

// GoOffline sets captain as unavailable
func (s *UserService) GoOffline(ctx context.Context, userID string) error {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	err = user.GoOffline()
	if err != nil {
		return err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return err
	}

	// Invalidate cache
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return nil
}

// UpdateLocation updates captain's current location
func (s *UserService) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	// Lightweight operation - update directly in repository
	err := s.repo.UpdateCaptainLocation(ctx, userID, lat, lng)
	if err != nil {
		return err
	}

	// Invalidate cache (location changed)
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return nil
}

// LinkPhone links a phone number to user account
func (s *UserService) LinkPhone(ctx context.Context, userID, phone string) (*aggregates.User, error) {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = user.AddPhoneNumber(phone)
	if err != nil {
		return nil, err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate and reload
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return loader.LoadUser(ctx, userID)
}

// ============================================================================
// ADMIN OPERATIONS
// ============================================================================

// ApproveKYC approves captain's KYC (admin only)
func (s *UserService) ApproveKYC(ctx context.Context, captainID string) (*aggregates.User, error) {
	user, err := s.GetUser(ctx, captainID)
	if err != nil {
		return nil, err
	}

	err = user.ApproveKyc()
	if err != nil {
		return nil, err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate and reload
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(captainID)

	return loader.LoadUser(ctx, captainID)
}

// RejectKYC rejects captain's KYC (admin only)
func (s *UserService) RejectKYC(ctx context.Context, captainID, reason string) (*aggregates.User, error) {
	user, err := s.GetUser(ctx, captainID)
	if err != nil {
		return nil, err
	}

	err = user.RejectKyc(reason)
	if err != nil {
		return nil, err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate and reload
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(captainID)

	return loader.LoadUser(ctx, captainID)
}

// BlockUser blocks a user account (admin only)
func (s *UserService) BlockUser(ctx context.Context, userID, reason string) (*aggregates.User, error) {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = user.Block(reason)
	if err != nil {
		return nil, err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate and reload
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return loader.LoadUser(ctx, userID)
}

// UnblockUser unblocks a user account (admin only)
func (s *UserService) UnblockUser(ctx context.Context, userID string) (*aggregates.User, error) {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = user.Unblock()
	if err != nil {
		return nil, err
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate and reload
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)

	return loader.LoadUser(ctx, userID)
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// InvalidateCache clears user from cache (useful after external updates)
func (s *UserService) InvalidateCache(ctx context.Context, userID string) {
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Clear(userID)
}

// PrimeCache adds a user to the cache (useful for optimization)
func (s *UserService) PrimeCache(ctx context.Context, user *aggregates.User) {
	loader := dataloader.MustGetUserLoader(ctx)
	loader.Prime(user.ID(), user)
}

// GetLoaderStats returns DataLoader statistics for monitoring
func (s *UserService) GetLoaderStats(ctx context.Context) dataloader.LoaderStats {
	loader := dataloader.MustGetUserLoader(ctx)
	return loader.Stats()
}
