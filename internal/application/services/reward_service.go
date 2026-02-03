package services

import (
	"context"
	"turtle/internal/domain"
)

// RewardService handles reward point allocation
type RewardService struct {
	penaltyService *PenaltyService
	orderRepo      domain.OrderRepository
}

// NewRewardService creates a new reward service
func NewRewardService(
	penaltyService *PenaltyService,
	orderRepo domain.OrderRepository,
) *RewardService {
	return &RewardService{
		penaltyService: penaltyService,
		orderRepo:      orderRepo,
	}
}

// ProcessOrderCompletion processes rewards after order completion
func (s *RewardService) ProcessOrderCompletion(
	ctx context.Context,
	orderID string,
	captainID string,
	customerRating float64,
	captainRating float64,
) error {
	// Award points to captain based on customer's rating
	captainPoints := s.calculateRatingPoints(captainRating)
	if captainPoints > 0 {
		if err := s.penaltyService.AddRewardPoints(
			ctx,
			captainID,
			captainPoints,
			"Order completed with rating",
		); err != nil {
			// Log error but don't fail
		}
	}

	// Award points to customer based on captain's rating of customer
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	customerPoints := s.calculateRatingPoints(customerRating)
	if customerPoints > 0 {
		if err := s.penaltyService.AddRewardPoints(
			ctx,
			order.BookedByUserID(),
			customerPoints,
			"Order completed with rating",
		); err != nil {
			// Log error but don't fail
		}
	}

	// Check milestone rewards
	s.checkMilestoneRewards(ctx, captainID)

	return nil
}

// calculateRatingPoints converts rating to reward points
func (s *RewardService) calculateRatingPoints(rating float64) int {
	switch {
	case rating >= 5.0:
		return 3
	case rating >= 4.0:
		return 2
	case rating >= 3.0:
		return 1
	default:
		return 0
	}
}

// checkMilestoneRewards checks and awards milestone rewards
func (s *RewardService) checkMilestoneRewards(
	ctx context.Context,
	userID string,
) error {
	// Get completed order count
	completedCount, err := s.orderRepo.CountCompletedByUser(ctx, userID)
	if err != nil {
		return err
	}

	// Milestone rewards
	var points int
	var reason string

	switch completedCount {
	case 10:
		points = 5
		reason = "Milestone: 10 completed orders"
	case 50:
		points = 10
		reason = "Milestone: 50 completed orders"
	case 100:
		points = 20
		reason = "Milestone: 100 completed orders"
	case 500:
		points = 50
		reason = "Milestone: 500 completed orders"
	case 1000:
		points = 100
		reason = "Milestone: 1000 completed orders"
	}

	if points > 0 {
		if err := s.penaltyService.AddRewardPoints(ctx, userID, points, reason); err != nil {
			return err
		}
	}

	// Check consecutive orders without cancellation
	// TODO: Implement streak tracking

	return nil
}
