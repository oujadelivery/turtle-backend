package services

import (
	"context"
	"errors"
	"time"
	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
)

// MatchingService handles captain-order matching
type MatchingService struct {
	orderRepo      domain.OrderRepository
	userRepo       domain.UserRepository
	assignmentRepo domain.CaptainAssignmentRepository
	orderService   *OrderService
}

// NewMatchingService creates a new matching service
func NewMatchingService(
	orderRepo domain.OrderRepository,
	userRepo domain.UserRepository,
	assignmentRepo domain.CaptainAssignmentRepository,
	orderService *OrderService,
) *MatchingService {
	return &MatchingService{
		orderRepo:      orderRepo,
		userRepo:       userRepo,
		assignmentRepo: assignmentRepo,
		orderService:   orderService,
	}
}

// FindNearbyAvailableCaptains finds captains near pickup location
func (s *MatchingService) FindNearbyAvailableCaptains(
	ctx context.Context,
	latitude, longitude float64,
	radiusKm float64,
) ([]*aggregates.User, error) {
	// Find captains within radius
	captains, err := s.userRepo.FindCaptainsNearby(ctx, latitude, longitude, radiusKm)
	if err != nil {
		return nil, err
	}

	// Filter by availability and KYC status
	available := make([]*aggregates.User, 0)
	for _, captain := range captains {
		if captain.IsCaptain() &&
			captain.CaptainProfile().IsAvailable() &&
			captain.CaptainProfile().KYCStatus() == "VERIFIED" &&
			captain.Status() == "ACTIVE" {
			available = append(available, captain)
		}
	}

	return available, nil
}

// AssignCaptainToOrder assigns a captain to an order
func (s *MatchingService) AssignCaptainToOrder(
	ctx context.Context,
	orderID string,
	captainID string,
	attemptNumber int,
) error {
	// 1. Load order
	order, err := s.orderService.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Assign captain (domain logic)
	if err := order.AssignCaptain(captainID, "SYSTEM"); err != nil {
		return err
	}

	// 3. Save order
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	// 4. Record assignment attempt
	timeout := time.Now().Add(2 * time.Minute) // 2 min to respond
	if err := s.assignmentRepo.CreateAttempt(ctx, orderID, captainID, attemptNumber, timeout); err != nil {
		return err
	}

	// 5. TODO: Send push notification to captain

	return nil
}

// StartMatching initiates the captain matching process
func (s *MatchingService) StartMatching(
	ctx context.Context,
	orderID string,
) error {
	// 1. Load order
	order, err := s.orderService.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Transition to searching state
	if err := order.StartCaptainSearch("SYSTEM"); err != nil {
		return err
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	// 3. Find nearby captains (start with 5km radius)
	pickup := order.PickupAddress()
	captains, err := s.FindNearbyAvailableCaptains(
		ctx,
		pickup.Location().Latitude(),
		pickup.Location().Longitude(),
		5.0, // 5km radius
	)
	if err != nil {
		return err
	}

	if len(captains) == 0 {
		return errors.New("no available captains found")
	}

	// 4. Try assigning to closest captain first
	// TODO: Implement smart matching (rating, acceptance rate, etc.)
	closestCaptain := captains[0]

	if err := s.AssignCaptainToOrder(ctx, orderID, closestCaptain.ID(), 1); err != nil {
		return err
	}

	return nil
}

// HandleCaptainTimeout handles captain not responding
func (s *MatchingService) HandleCaptainTimeout(
	ctx context.Context,
	orderID string,
	captainID string,
) error {
	// 1. Mark attempt as timeout
	if err := s.assignmentRepo.MarkTimeout(ctx, orderID, captainID); err != nil {
		return err
	}

	// 2. Get order and decline captain
	order, err := s.orderService.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	if err := order.CaptainDecline(captainID, "Timeout - no response"); err != nil {
		return err
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	// 3. Try next captain (expand radius if needed)
	// TODO: Implement retry logic with expanding radius

	return nil
}
