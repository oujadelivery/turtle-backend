package services

import (
	"context"
	"errors"
	"time"
	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
	"turtle/internal/domain/events"
	"turtle/internal/domain/valueobjects"
	"turtle/internal/infrastructure/dataloader"

	"github.com/google/uuid"
)

// OrderService handles order operations with DataLoader integration
type OrderService struct {
	orderRepo domain.OrderRepository
	userRepo  domain.UserRepository
	eventBus  EventBus // For publishing domain events
}

// NewOrderService creates a new order service
func NewOrderService(
	orderRepo domain.OrderRepository,
	userRepo domain.UserRepository,
	eventBus EventBus,
) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
		userRepo:  userRepo,
		eventBus:  eventBus,
	}
}

// ============================================================================
// CORE OPERATIONS
// ============================================================================

// CreateOrder creates a new order
func (s *OrderService) CreateOrder(
	ctx context.Context,
	input CreateOrderInput,
) (*aggregates.Order, error) {
	// 1. Validate booker exists and has proper role
	booker, err := s.userRepo.FindByID(ctx, input.BookedByUserID)
	if err != nil {
		return nil, errors.New("booker not found")
	}

	if !booker.IsCustomer() {
		return nil, errors.New("only customers can create orders")
	}

	// 2. Business Rule: Check if user is currently acting as captain
	if booker.IsCaptain() {
		// Check if captain has active orders
		activeOrders, err := s.orderRepo.FindActiveCaptainOrders(ctx, booker.ID())
		if err != nil {
			return nil, err
		}
		if len(activeOrders) > 0 {
			return nil, errors.New("cannot create order while actively delivering as captain")
		}
	}

	// 3. Create value objects
	sender, err := valueobjects.NewDeliveryContact(
		input.SenderName,
		input.SenderPhone,
		input.SenderAlternatePhone,
		input.SenderNotes,
	)
	if err != nil {
		return nil, err
	}

	receiver, err := valueobjects.NewDeliveryContact(
		input.ReceiverName,
		input.ReceiverPhone,
		input.ReceiverAlternatePhone,
		input.ReceiverNotes,
	)
	if err != nil {
		return nil, err
	}

	pickupLocation, err := valueobjects.NewLocation(
		input.PickupLatitude,
		input.PickupLongitude,
	)
	if err != nil {
		return nil, err
	}

	pickupAddress, err := valueobjects.NewOrderAddress(
		input.PickupAddressLine1,
		input.PickupAddressLine2,
		input.PickupLandmark,
		input.PickupCity,
		input.PickupState,
		input.PickupPostalCode,
		pickupLocation,
	)
	if err != nil {
		return nil, err
	}

	deliveryLocation, err := valueobjects.NewLocation(
		input.DeliveryLatitude,
		input.DeliveryLongitude,
	)
	if err != nil {
		return nil, err
	}

	deliveryAddress, err := valueobjects.NewOrderAddress(
		input.DeliveryAddressLine1,
		input.DeliveryAddressLine2,
		input.DeliveryLandmark,
		input.DeliveryCity,
		input.DeliveryState,
		input.DeliveryPostalCode,
		deliveryLocation,
	)
	if err != nil {
		return nil, err
	}

	// Parcel details
	var parcel *valueobjects.Parcel
	if input.OrderType == aggregates.OrderTypeParcel {
		parcelValue, _ := valueobjects.NewMoney(input.ParcelValuePaise, "INR")
		parcel, err = valueobjects.NewParcel(
			valueobjects.ParcelType(input.ParcelType),
			input.ParcelWeight,
			input.ParcelDescription,
			parcelValue,
			input.ParcelImages,
		)
		if err != nil {
			return nil, err
		}
	}

	// 4. Create order aggregate
	order, err := aggregates.NewOrder(
		uuid.New().String(),
		input.OrderType,
		input.BookedByUserID,
		sender,
		receiver,
		pickupAddress,
		deliveryAddress,
		parcel,
		input.PaymentMethod,
		input.CustomerNotes,
		input.ScheduledPickupTime,
	)
	if err != nil {
		return nil, err
	}

	// 5. Save order
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// 6. Publish domain event
	s.eventBus.Publish(events.OrderCreated{
		OrderID:          order.ID(),
		OrderType:        order.OrderType(),
		BookedByUserID:   order.BookedByUserID(),
		PickupLocation:   pickupAddress.FormattedAddress(),
		DeliveryLocation: deliveryAddress.FormattedAddress(),
		OccurredAt_:      time.Now(),
	})

	return order, nil
}

// GetOrder retrieves an order by ID (with DataLoader)
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*aggregates.Order, error) {
	loader := dataloader.MustGetOrderLoader(ctx)
	return loader.LoadOrder(ctx, orderID)
}

// GetUserOrders retrieves all orders for a user
func (s *OrderService) GetUserOrders(
	ctx context.Context,
	userID string,
	limit, offset int,
) ([]*aggregates.Order, int64, error) {
	return s.orderRepo.FindByUserID(ctx, userID, limit, offset)
}

// GetCaptainOrders retrieves orders assigned to captain
func (s *OrderService) GetCaptainOrders(
	ctx context.Context,
	captainID string,
	limit, offset int,
) ([]*aggregates.Order, int64, error) {
	return s.orderRepo.FindByCaptainID(ctx, captainID, limit, offset)
}

// ============================================================================
// STATE TRANSITIONS
// ============================================================================

// CaptainAcceptOrder handles captain accepting an order
func (s *OrderService) CaptainAcceptOrder(
	ctx context.Context,
	orderID string,
	captainUserID string,
) (*aggregates.Order, error) {
	// 1. Load order
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// 2. Verify captain identity
	captain, err := s.userRepo.FindByID(ctx, captainUserID)
	if err != nil {
		return nil, errors.New("captain not found")
	}

	if !captain.IsCaptain() {
		return nil, errors.New("user is not a captain")
	}

	// 3. Domain logic: Captain accepts
	if err := order.CaptainAccept(captainUserID); err != nil {
		return nil, err
	}

	// 4. Generate OTPs
	if err := order.GenerateOTPs(); err != nil {
		return nil, err
	}

	// 5. Save with optimistic locking
	if err := s.orderRepo.Update(ctx, order); err != nil {
		if err == domain.ErrConcurrentModification {
			return nil, errors.New("order was modified by another request, please retry")
		}
		return nil, err
	}

	// 6. Publish event
	s.eventBus.Publish(events.CaptainAssigned{
		OrderID:     order.ID(),
		CaptainID:   captainUserID,
		OccurredAt_: time.Now(),
	})

	// 7. Clear cache and reload
	s.InvalidateOrderCache(ctx, orderID)
	return s.GetOrder(ctx, orderID)
}

// ConfirmPickup confirms parcel pickup with OTP
func (s *OrderService) ConfirmPickup(
	ctx context.Context,
	orderID string,
	captainUserID string,
	otp string,
) (*aggregates.Order, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := order.ConfirmPickup(captainUserID, otp); err != nil {
		return nil, err
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	s.eventBus.Publish(events.OrderPickedUp{
		OrderID:     order.ID(),
		CaptainID:   captainUserID,
		PickupTime:  *order.ActualPickupTime(),
		OccurredAt_: time.Now(),
	})

	s.InvalidateOrderCache(ctx, orderID)
	return s.GetOrder(ctx, orderID)
}

// ConfirmDelivery confirms parcel delivery with OTP
func (s *OrderService) ConfirmDelivery(
	ctx context.Context,
	orderID string,
	captainUserID string,
	otp string,
) (*aggregates.Order, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := order.ConfirmDelivery(captainUserID, otp); err != nil {
		return nil, err
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	s.eventBus.Publish(events.OrderDelivered{
		OrderID:      order.ID(),
		CaptainID:    captainUserID,
		DeliveryTime: *order.ActualDeliveryTime(),
		OccurredAt_:  time.Now(),
	})

	s.InvalidateOrderCache(ctx, orderID)
	return s.GetOrder(ctx, orderID)
}

// CompleteOrder marks order as completed
func (s *OrderService) CompleteOrder(
	ctx context.Context,
	orderID string,
	triggeredBy string,
) (*aggregates.Order, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := order.Complete(triggeredBy); err != nil {
		return nil, err
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	s.eventBus.Publish(events.OrderCompleted{
		OrderID:        order.ID(),
		CaptainID:      *order.AssignedCaptainID(),
		TotalAmount:    order.Pricing().Total().AmountPaise(),
		CaptainEarning: order.Pricing().CaptainEarning().AmountPaise(),
		OccurredAt_:    time.Now(),
	})

	s.InvalidateOrderCache(ctx, orderID)
	return s.GetOrder(ctx, orderID)
}

// ============================================================================
// CANCELLATION
// ============================================================================

// CancelOrder cancels an order with proper validation
func (s *OrderService) CancelOrder(
	ctx context.Context,
	orderID string,
	cancelledBy string,
	cancelledByRole string,
	reason string,
) (*aggregates.Order, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Calculate refund and penalty
	// TODO: Implement refund/penalty calculation logic
	refundAmount, _ := valueobjects.NewMoney(0, "INR")
	penaltyAmount, _ := valueobjects.NewMoney(0, "INR")

	if err := order.Cancel(cancelledBy, cancelledByRole, reason, refundAmount, penaltyAmount); err != nil {
		return nil, err
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	s.eventBus.Publish(events.OrderCancelled{
		OrderID:         order.ID(),
		CancelledBy:     cancelledBy,
		CancelledByRole: cancelledByRole,
		Reason:          reason,
		OccurredAt_:     time.Now(),
	})

	s.InvalidateOrderCache(ctx, orderID)
	return s.GetOrder(ctx, orderID)
}

// ============================================================================
// CACHE MANAGEMENT
// ============================================================================

// InvalidateOrderCache clears order from cache
func (s *OrderService) InvalidateOrderCache(ctx context.Context, orderID string) {
	loader := dataloader.MustGetOrderLoader(ctx)
	loader.Clear(orderID)
}

// PrimeCache adds order to cache
func (s *OrderService) PrimeCache(ctx context.Context, order *aggregates.Order) {
	loader := dataloader.MustGetOrderLoader(ctx)
	loader.Prime(order.ID(), order)
}

// ============================================================================
// INPUT TYPES
// ============================================================================

type CreateOrderInput struct {
	OrderType      aggregates.OrderType
	BookedByUserID string

	// Sender
	SenderName           string
	SenderPhone          string
	SenderAlternatePhone *string
	SenderNotes          string

	// Receiver
	ReceiverName           string
	ReceiverPhone          string
	ReceiverAlternatePhone *string
	ReceiverNotes          string

	// Pickup Address
	PickupAddressLine1 string
	PickupAddressLine2 string
	PickupLandmark     string
	PickupCity         string
	PickupState        string
	PickupPostalCode   string
	PickupLatitude     float64
	PickupLongitude    float64

	// Delivery Address
	DeliveryAddressLine1 string
	DeliveryAddressLine2 string
	DeliveryLandmark     string
	DeliveryCity         string
	DeliveryState        string
	DeliveryPostalCode   string
	DeliveryLatitude     float64
	DeliveryLongitude    float64

	// Parcel
	ParcelType        string
	ParcelWeight      float64
	ParcelDescription string
	ParcelValuePaise  int64
	ParcelImages      []string

	// Payment
	PaymentMethod aggregates.PaymentMethod

	// Notes
	CustomerNotes string

	// Timing
	ScheduledPickupTime *time.Time
}

// EventBus interface for publishing domain events
type EventBus interface {
	Publish(event events.DomainEvent)
}
