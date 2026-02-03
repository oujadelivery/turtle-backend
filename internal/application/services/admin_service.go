package services

import (
	"context"
	"errors"
	"turtle/internal/domain"
	pkgErrors "turtle/pkg/errors"
)

// AdminService provides admin override capabilities
type AdminService struct {
	cancellationService *CancellationService
	penaltyService      *PenaltyService
	orderRepo           domain.OrderRepository
	userRepo            domain.UserRepository
}

// NewAdminService creates a new admin service
func NewAdminService(
	cancellationService *CancellationService,
	penaltyService *PenaltyService,
	orderRepo domain.OrderRepository,
	userRepo domain.UserRepository,
) *AdminService {
	return &AdminService{
		cancellationService: cancellationService,
		penaltyService:      penaltyService,
		orderRepo:           orderRepo,
		userRepo:            userRepo,
	}
}

// AdminCancelOrder cancels order with admin override
func (s *AdminService) AdminCancelOrder(
	ctx context.Context,
	adminID string,
	orderID string,
	reason string,
	noPenalty bool,
) (*CancelOrderResult, error) {
	// Verify admin
	admin, err := s.userRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, err
	}

	if !admin.IsAdmin() {
		return nil, pkgErrors.ErrUnauthorized("Only admins can perform this action")
	}

	// Cancel with admin override
	result, err := s.cancellationService.CancelOrder(ctx, CancelOrderInput{
		OrderID:         orderID,
		CancelledBy:     adminID,
		CancelledByRole: "ADMIN",
		Reason:          reason,
		AdminOverride:   noPenalty,
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// ViewUserPenaltyDetails retrieves detailed penalty information
func (s *AdminService) ViewUserPenaltyDetails(
	ctx context.Context,
	adminID string,
	userID string,
) (*UserPenaltyDetails, error) {
	// Verify admin
	admin, err := s.userRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, err
	}

	if !admin.IsAdmin() {
		return nil, pkgErrors.ErrUnauthorized("Only admins can view penalty details")
	}

	// Get penalty record
	penalty, err := s.penaltyService.GetUserPenalty(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get cancellation history
	cancellations, _, err := s.cancellationService.GetCancellationHistory(ctx, userID, 100, 0)
	if err != nil {
		return nil, err
	}

	// Get cancellation stats
	stats, err := s.cancellationService.GetCancellationStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserPenaltyDetails{
		Penalty:             penalty,
		CancellationHistory: cancellations,
		CancellationStats:   stats,
	}, nil
}

// ResetUserPenalty resets penalty points for a user
func (s *AdminService) ResetUserPenalty(
	ctx context.Context,
	adminID string,
	userID string,
	reason string,
) error {
	// Verify admin
	admin, err := s.userRepo.FindByID(ctx, adminID)
	if err != nil {
		return err
	}

	if !admin.IsAdmin() {
		return pkgErrors.ErrUnauthorized("Only admins can reset penalties")
	}

	return s.penaltyService.ResetPenaltyPoints(ctx, userID, adminID, reason)
}

// AdjustUserPoints manually adjusts user points
func (s *AdminService) AdjustUserPoints(
	ctx context.Context,
	adminID string,
	userID string,
	penaltyAdjustment int,
	rewardAdjustment int,
	reason string,
) error {
	// Verify admin
	admin, err := s.userRepo.FindByID(ctx, adminID)
	if err != nil {
		return err
	}

	if !admin.IsAdmin() {
		return pkgErrors.ErrUnauthorized("Only admins can adjust points")
	}

	if reason == "" {
		return errors.New("reason is required for manual point adjustment")
	}

	return s.penaltyService.AdjustPoints(
		ctx,
		userID,
		penaltyAdjustment,
		rewardAdjustment,
		adminID,
		reason,
	)
}

// SuspendUser suspends a user manually
func (s *AdminService) SuspendUser(
	ctx context.Context,
	adminID string,
	userID string,
	reason string,
	durationHours int,
) error {
	// Verify admin
	admin, err := s.userRepo.FindByID(ctx, adminID)
	if err != nil {
		return err
	}

	if !admin.IsAdmin() {
		return pkgErrors.ErrUnauthorized("Only admins can suspend users")
	}

	// Get user
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Update user status
	if err := user.Suspend(reason); err != nil {
		return err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// TODO: Log admin action
	// TODO: Send notification to user

	return nil
}

// UnsuspendUser removes suspension from a user
func (s *AdminService) UnsuspendUser(
	ctx context.Context,
	adminID string,
	userID string,
	reason string,
) error {
	// Verify admin
	admin, err := s.userRepo.FindByID(ctx, adminID)
	if err != nil {
		return err
	}

	if !admin.IsAdmin() {
		return pkgErrors.ErrUnauthorized("Only admins can unsuspend users")
	}

	// Get user
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Activate user
	if err := user.Activate(); err != nil {
		return err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Also clear penalty suspension
	penalty, err := s.penaltyService.GetUserPenalty(ctx, userID)
	if err == nil && penalty.IsSuspended {
		penalty.IsSuspended = false
		penalty.SuspensionUntil = nil
		// TODO: Update penalty record
	}

	// TODO: Log admin action
	// TODO: Send notification to user

	return nil
}

type UserPenaltyDetails struct {
	Penalty             *domain.UserPenalty
	CancellationHistory []*domain.CancellationEvent
	CancellationStats   *CancellationStats
}
