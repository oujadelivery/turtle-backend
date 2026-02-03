package services

import (
	"context"
	"time"
	"turtle/internal/domain"
)

// PenaltyService manages user penalty and reward points
type PenaltyService struct {
	penaltyRepo domain.UserPenaltyRepository
	userRepo    domain.UserRepository
}

// NewPenaltyService creates a new penalty service
func NewPenaltyService(
	penaltyRepo domain.UserPenaltyRepository,
	userRepo domain.UserRepository,
) *PenaltyService {
	return &PenaltyService{
		penaltyRepo: penaltyRepo,
		userRepo:    userRepo,
	}
}

// AddPenaltyPoints adds penalty points to a user
func (s *PenaltyService) AddPenaltyPoints(
	ctx context.Context,
	userID string,
	points int,
	reason string,
) error {
	if points <= 0 {
		return nil
	}

	// Get or create penalty record
	penalty, err := s.penaltyRepo.FindByUserID(ctx, userID)
	if err == domain.ErrNotFound {
		penalty = &domain.UserPenalty{
			UserID:             userID,
			TotalPenaltyPoints: 0,
			TotalRewardPoints:  0,
			NetScore:           0,
			IsSuspended:        false,
		}
	} else if err != nil {
		return err
	}

	// Add penalty points
	penalty.TotalPenaltyPoints += points
	penalty.NetScore = penalty.TotalRewardPoints - penalty.TotalPenaltyPoints
	penalty.LastPenaltyAt = timePtr(time.Now())

	// Check if suspension is needed
	if err := s.checkAndApplySuspension(penalty); err != nil {
		return err
	}

	// Save penalty record
	if penalty.ID == 0 {
		if err := s.penaltyRepo.Create(ctx, penalty); err != nil {
			return err
		}
	} else {
		if err := s.penaltyRepo.Update(ctx, penalty); err != nil {
			return err
		}
	}

	// TODO: Send notification about penalty points

	return nil
}

// AddRewardPoints adds reward points to a user
func (s *PenaltyService) AddRewardPoints(
	ctx context.Context,
	userID string,
	points int,
	reason string,
) error {
	if points <= 0 {
		return nil
	}

	// Get or create penalty record
	penalty, err := s.penaltyRepo.FindByUserID(ctx, userID)
	if err == domain.ErrNotFound {
		penalty = &domain.UserPenalty{
			UserID:             userID,
			TotalPenaltyPoints: 0,
			TotalRewardPoints:  0,
			NetScore:           0,
			IsSuspended:        false,
		}
	} else if err != nil {
		return err
	}

	// Add reward points
	penalty.TotalRewardPoints += points
	penalty.NetScore = penalty.TotalRewardPoints - penalty.TotalPenaltyPoints
	penalty.LastRewardAt = timePtr(time.Now())

	// Check if suspension can be lifted
	if penalty.IsSuspended && penalty.NetScore >= 0 {
		penalty.IsSuspended = false
		penalty.SuspensionUntil = nil
	}

	// Save penalty record
	if penalty.ID == 0 {
		if err := s.penaltyRepo.Create(ctx, penalty); err != nil {
			return err
		}
	} else {
		if err := s.penaltyRepo.Update(ctx, penalty); err != nil {
			return err
		}
	}

	// TODO: Send notification about reward points

	return nil
}

// checkAndApplySuspension applies suspension based on penalty points
func (s *PenaltyService) checkAndApplySuspension(penalty *domain.UserPenalty) error {
	points := penalty.TotalPenaltyPoints

	var suspensionDuration time.Duration

	switch {
	case points >= 51:
		// Permanent suspension - needs admin review
		penalty.IsSuspended = true
		penalty.SuspensionUntil = nil
		return nil

	case points >= 31:
		// 7-day suspension
		suspensionDuration = 7 * 24 * time.Hour

	case points >= 21:
		// 24-hour suspension
		suspensionDuration = 24 * time.Hour

	case points >= 11:
		// Warning only, no suspension
		// TODO: Send warning notification
		return nil

	default:
		// No action needed
		return nil
	}

	until := time.Now().Add(suspensionDuration)
	penalty.IsSuspended = true
	penalty.SuspensionUntil = &until

	return nil
}

// GetUserPenalty retrieves penalty record for a user
func (s *PenaltyService) GetUserPenalty(
	ctx context.Context,
	userID string,
) (*domain.UserPenalty, error) {
	penalty, err := s.penaltyRepo.FindByUserID(ctx, userID)
	if err == domain.ErrNotFound {
		// Return empty penalty record
		return &domain.UserPenalty{
			UserID:             userID,
			TotalPenaltyPoints: 0,
			TotalRewardPoints:  0,
			NetScore:           0,
			IsSuspended:        false,
		}, nil
	}
	return penalty, err
}

// CheckSuspensionStatus checks if user is currently suspended
func (s *PenaltyService) CheckSuspensionStatus(
	ctx context.Context,
	userID string,
) (bool, *time.Time, error) {
	penalty, err := s.GetUserPenalty(ctx, userID)
	if err != nil {
		return false, nil, err
	}

	// Check if suspension has expired
	if penalty.IsSuspended && penalty.SuspensionUntil != nil {
		if time.Now().After(*penalty.SuspensionUntil) {
			// Lift suspension
			penalty.IsSuspended = false
			penalty.SuspensionUntil = nil
			if err := s.penaltyRepo.Update(ctx, penalty); err != nil {
				return false, nil, err
			}
			return false, nil, nil
		}
	}

	return penalty.IsSuspended, penalty.SuspensionUntil, nil
}

// ResetPenaltyPoints resets penalty points (admin only)
func (s *PenaltyService) ResetPenaltyPoints(
	ctx context.Context,
	userID string,
	adminID string,
	reason string,
) error {
	penalty, err := s.penaltyRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	penalty.TotalPenaltyPoints = 0
	penalty.NetScore = penalty.TotalRewardPoints
	penalty.IsSuspended = false
	penalty.SuspensionUntil = nil

	if err := s.penaltyRepo.Update(ctx, penalty); err != nil {
		return err
	}

	// TODO: Log admin action

	return nil
}

// AdjustPoints manually adjusts points (admin only)
func (s *PenaltyService) AdjustPoints(
	ctx context.Context,
	userID string,
	penaltyAdjustment int,
	rewardAdjustment int,
	adminID string,
	reason string,
) error {
	penalty, err := s.penaltyRepo.FindByUserID(ctx, userID)
	if err == domain.ErrNotFound {
		penalty = &domain.UserPenalty{
			UserID:             userID,
			TotalPenaltyPoints: 0,
			TotalRewardPoints:  0,
			NetScore:           0,
		}
	} else if err != nil {
		return err
	}

	// Apply adjustments
	penalty.TotalPenaltyPoints += penaltyAdjustment
	penalty.TotalRewardPoints += rewardAdjustment

	// Ensure non-negative
	if penalty.TotalPenaltyPoints < 0 {
		penalty.TotalPenaltyPoints = 0
	}
	if penalty.TotalRewardPoints < 0 {
		penalty.TotalRewardPoints = 0
	}

	penalty.NetScore = penalty.TotalRewardPoints - penalty.TotalPenaltyPoints

	// Save
	if penalty.ID == 0 {
		if err := s.penaltyRepo.Create(ctx, penalty); err != nil {
			return err
		}
	} else {
		if err := s.penaltyRepo.Update(ctx, penalty); err != nil {
			return err
		}
	}

	// TODO: Log admin action

	return nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}
