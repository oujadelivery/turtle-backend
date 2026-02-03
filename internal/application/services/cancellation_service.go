package services

import (
	"context"
	"time"
	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
	"turtle/internal/domain/valueobjects"
	pkgErrors "turtle/pkg/errors"
)

// CancellationService handles order cancellation logic
type CancellationService struct {
	orderRepo        domain.OrderRepository
	cancellationRepo domain.CancellationEventRepository
	penaltyService   *PenaltyService
	paymentService   *PaymentService
}

// NewCancellationService creates a new cancellation service
func NewCancellationService(
	orderRepo domain.OrderRepository,
	cancellationRepo domain.CancellationEventRepository,
	penaltyService *PenaltyService,
	paymentService *PaymentService,
) *CancellationService {
	return &CancellationService{
		orderRepo:        orderRepo,
		cancellationRepo: cancellationRepo,
		penaltyService:   penaltyService,
		paymentService:   paymentService,
	}
}

// CancelOrderInput contains cancellation request details
type CancelOrderInput struct {
	OrderID         string
	CancelledBy     string
	CancelledByRole string // CUSTOMER, CAPTAIN, ADMIN, SYSTEM
	Reason          string
	AdminOverride   bool // Admin can override penalty rules
}

// CancelOrderResult contains cancellation outcome
type CancelOrderResult struct {
	Order           *aggregates.Order
	RefundAmount    *valueobjects.Money
	PenaltyAmount   *valueobjects.Money
	PenaltyPoints   int
	WasAutoRefunded bool
}

// CancelOrder processes order cancellation with full business logic
func (s *CancellationService) CancelOrder(
	ctx context.Context,
	input CancelOrderInput,
) (*CancelOrderResult, error) {
	// 1. Load order
	order, err := s.orderRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, pkgErrors.ErrOrderNotFound(input.OrderID)
	}

	// 2. Validate cancellation is allowed
	if err := s.validateCancellation(order, input); err != nil {
		return nil, err
	}

	// 3. Calculate refund and penalty
	refundCalc := s.calculateRefund(order, input)
	penaltyCalc := s.calculatePenalty(order, input)

	// 4. Apply admin override if applicable
	if input.AdminOverride && input.CancelledByRole == "ADMIN" {
		penaltyCalc.PenaltyPoints = 0
		penaltyCalc.PenaltyAmount = valueobjects.Zero("INR")
	}

	// 5. Cancel order in domain
	if err := order.Cancel(
		input.CancelledBy,
		input.CancelledByRole,
		input.Reason,
		refundCalc.Amount,
		penaltyCalc.PenaltyAmount,
	); err != nil {
		return nil, err
	}

	// 6. Save order
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	// 7. Record cancellation event
	cancellationEvent := &domain.CancellationEvent{
		OrderID:                  input.OrderID,
		UserID:                   input.CancelledBy,
		CancelledByRole:          input.CancelledByRole,
		OrderStateAtCancellation: string(order.State()),
		Reason:                   input.Reason,
		RefundAmount:             refundCalc.Amount.AmountPaise(),
		PenaltyAmount:            penaltyCalc.PenaltyAmount.AmountPaise(),
		PenaltyPoints:            penaltyCalc.PenaltyPoints,
		CreatedAt:                time.Now(),
	}

	if err := s.cancellationRepo.Create(ctx, cancellationEvent); err != nil {
		// Log error but don't fail the cancellation
		// TODO: Implement proper logging
	}

	// 8. Apply penalty points
	if penaltyCalc.PenaltyPoints > 0 {
		if err := s.penaltyService.AddPenaltyPoints(
			ctx,
			input.CancelledBy,
			penaltyCalc.PenaltyPoints,
			"Order cancellation: "+input.Reason,
		); err != nil {
			// Log error but don't fail
		}
	}

	// 9. Process refund if applicable
	var wasAutoRefunded bool
	if refundCalc.Amount != nil && !refundCalc.Amount.IsZero() {
		if err := s.paymentService.ProcessRefund(
			ctx,
			order,
			refundCalc.Amount,
			"Order cancellation refund",
		); err != nil {
			// Log error - refund will need manual processing
			wasAutoRefunded = false
		} else {
			wasAutoRefunded = true
		}
	}

	return &CancelOrderResult{
		Order:           order,
		RefundAmount:    refundCalc.Amount,
		PenaltyAmount:   penaltyCalc.PenaltyAmount,
		PenaltyPoints:   penaltyCalc.PenaltyPoints,
		WasAutoRefunded: wasAutoRefunded,
	}, nil
}

// validateCancellation checks if cancellation is allowed
func (s *CancellationService) validateCancellation(
	order *aggregates.Order,
	input CancelOrderInput,
) error {
	// Terminal states cannot be cancelled
	if order.IsTerminalState() {
		return pkgErrors.ErrOrderAlreadyCompleted
	}

	state := order.State()

	// Admin can cancel anytime
	if input.CancelledByRole == "ADMIN" || input.CancelledByRole == "SYSTEM" {
		return nil
	}

	// After pickup, only admin can cancel
	if state == aggregates.OrderStatePickedUp ||
		state == aggregates.OrderStateInTransit ||
		state == aggregates.OrderStateAtDelivery {
		return pkgErrors.ErrCannotCancelAfterPickup
	}

	// Customer cancellation validation
	if input.CancelledByRole == "CUSTOMER" {
		if input.CancelledBy != order.BookedByUserID() {
			return pkgErrors.ErrUnauthorized("Only the booker can cancel this order")
		}
	}

	// Captain cancellation validation
	if input.CancelledByRole == "CAPTAIN" {
		if order.AssignedCaptainID() == nil {
			return pkgErrors.ErrCaptainNotAssigned
		}
		if *order.AssignedCaptainID() != input.CancelledBy {
			return pkgErrors.ErrUnauthorized("Only assigned captain can cancel")
		}
		// Captain can only cancel before en route
		if state != aggregates.OrderStateCaptainAssigned &&
			state != aggregates.OrderStateCaptainAccepted {
			return pkgErrors.ErrCannotCancelAfterPickup
		}
	}

	return nil
}

// RefundCalculation contains refund calculation result
type RefundCalculation struct {
	Amount           *valueobjects.Money
	RefundPercentage int
	Reason           string
}

// calculateRefund determines refund amount based on state and timing
func (s *CancellationService) calculateRefund(
	order *aggregates.Order,
	input CancelOrderInput,
) RefundCalculation {
	state := order.State()
	total := order.Pricing().Total()
	currency := total.Currency()

	// Admin override - full refund
	if input.AdminOverride && input.CancelledByRole == "ADMIN" {
		return RefundCalculation{
			Amount:           total,
			RefundPercentage: 100,
			Reason:           "Admin override - full refund",
		}
	}

	// Before captain assignment - 100% refund
	if state == aggregates.OrderStateCreated ||
		state == aggregates.OrderStatePriceEstimated ||
		state == aggregates.OrderStateSearchingCaptain {
		return RefundCalculation{
			Amount:           total,
			RefundPercentage: 100,
			Reason:           "Cancelled before captain assignment",
		}
	}

	// Captain assigned but not accepted - 100% refund
	if state == aggregates.OrderStateCaptainAssigned {
		return RefundCalculation{
			Amount:           total,
			RefundPercentage: 100,
			Reason:           "Cancelled before captain accepted",
		}
	}

	// Captain accepted - check timing
	if state == aggregates.OrderStateCaptainAccepted {
		// Free cancellation window (5 minutes)
		acceptedTime := order.StateUpdatedAt()
		elapsed := time.Since(acceptedTime)

		if elapsed <= 5*time.Minute {
			return RefundCalculation{
				Amount:           total,
				RefundPercentage: 100,
				Reason:           "Cancelled within free cancellation window",
			}
		}

		// After free window - 80% refund
		refundAmount, _ := total.Multiply(0.80)
		return RefundCalculation{
			Amount:           refundAmount,
			RefundPercentage: 80,
			Reason:           "Cancelled after free cancellation window",
		}
	}

	// Captain en route - 50% refund
	if state == aggregates.OrderStateCaptainEnRoute {
		refundAmount, _ := total.Multiply(0.50)
		return RefundCalculation{
			Amount:           refundAmount,
			RefundPercentage: 50,
			Reason:           "Cancelled after captain started journey",
		}
	}

	// After pickup - no refund (should be admin only)
	return RefundCalculation{
		Amount:           valueobjects.Zero(currency),
		RefundPercentage: 0,
		Reason:           "No refund after pickup",
	}
}

// PenaltyCalculation contains penalty calculation result
type PenaltyCalculation struct {
	PenaltyPoints int
	PenaltyAmount *valueobjects.Money
	Reason        string
}

// calculatePenalty determines penalty points and amount
func (s *CancellationService) calculatePenalty(
	order *aggregates.Order,
	input CancelOrderInput,
) PenaltyCalculation {
	state := order.State()
	currency := order.Pricing().Total().Currency()

	// No penalty for system cancellations
	if input.CancelledByRole == "SYSTEM" {
		return PenaltyCalculation{
			PenaltyPoints: 0,
			PenaltyAmount: valueobjects.Zero(currency),
			Reason:        "System cancellation - no penalty",
		}
	}

	// Define penalty matrix
	var points int
	var reason string

	switch state {
	case aggregates.OrderStateCreated,
		aggregates.OrderStatePriceEstimated,
		aggregates.OrderStateSearchingCaptain:
		points = 0
		reason = "No penalty before captain assignment"

	case aggregates.OrderStateCaptainAssigned:
		if input.CancelledByRole == "CUSTOMER" {
			points = 1
			reason = "Customer cancelled after captain assignment"
		} else if input.CancelledByRole == "CAPTAIN" {
			points = 2
			reason = "Captain declined assigned order"
		}

	case aggregates.OrderStateCaptainAccepted:
		acceptedTime := order.StateUpdatedAt()
		elapsed := time.Since(acceptedTime)

		if input.CancelledByRole == "CUSTOMER" {
			if elapsed <= 5*time.Minute {
				points = 0
				reason = "Customer cancelled within free window"
			} else {
				points = 2
				reason = "Customer cancelled after free window"
			}
		} else if input.CancelledByRole == "CAPTAIN" {
			points = 5
			reason = "Captain cancelled after accepting"
		}

	case aggregates.OrderStateCaptainEnRoute:
		if input.CancelledByRole == "CUSTOMER" {
			points = 5
			reason = "Customer cancelled after captain started journey"
		} else if input.CancelledByRole == "CAPTAIN" {
			points = 10
			reason = "Captain cancelled after starting journey"
		}

	case aggregates.OrderStatePickedUp,
		aggregates.OrderStateInTransit:
		// Should only happen via admin
		if input.CancelledByRole == "CUSTOMER" {
			points = 10
			reason = "Customer cancelled after pickup"
		} else if input.CancelledByRole == "CAPTAIN" {
			points = 15
			reason = "Captain cancelled after pickup"
		}

	case aggregates.OrderStateAtDelivery:
		if input.CancelledByRole == "CUSTOMER" {
			points = 15
			reason = "Customer cancelled at delivery location"
		} else if input.CancelledByRole == "CAPTAIN" {
			points = 20
			reason = "Captain cancelled at delivery location"
		}

	default:
		points = 0
		reason = "No penalty applicable"
	}

	// Penalty amount could be used for future penalty charges
	// For now, set to zero (penalties are just points)
	penaltyAmount := valueobjects.Zero(currency)

	return PenaltyCalculation{
		PenaltyPoints: points,
		PenaltyAmount: penaltyAmount,
		Reason:        reason,
	}
}

// GetCancellationHistory retrieves user's cancellation history
func (s *CancellationService) GetCancellationHistory(
	ctx context.Context,
	userID string,
	limit, offset int,
) ([]*domain.CancellationEvent, int64, error) {
	return s.cancellationRepo.FindByUserID(ctx, userID, limit, offset)
}

// GetCancellationStats retrieves cancellation statistics
func (s *CancellationService) GetCancellationStats(
	ctx context.Context,
	userID string,
) (*CancellationStats, error) {
	events, _, err := s.cancellationRepo.FindByUserID(ctx, userID, 1000, 0)
	if err != nil {
		return nil, err
	}

	stats := &CancellationStats{
		TotalCancellations: len(events),
	}

	for _, event := range events {
		stats.TotalPenaltyPoints += event.PenaltyPoints

		switch event.CancelledByRole {
		case "CUSTOMER":
			stats.CustomerCancellations++
		case "CAPTAIN":
			stats.CaptainCancellations++
		}

		// Cancellations in last 30 days
		if time.Since(event.CreatedAt) <= 30*24*time.Hour {
			stats.RecentCancellations++
		}
	}

	return stats, nil
}

type CancellationStats struct {
	TotalCancellations    int
	CustomerCancellations int
	CaptainCancellations  int
	RecentCancellations   int // Last 30 days
	TotalPenaltyPoints    int
}
