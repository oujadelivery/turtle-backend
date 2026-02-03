package aggregates

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"turtle/internal/domain/valueobjects"
)

// Order is the aggregate root for all order-related operations
type Order struct {
	// Identity
	id      string
	version int // Optimistic locking

	// Order Type (future-proof)
	orderType OrderType // PARCEL, RIDE, MULTI_STOP, SCHEDULED

	// Parties (Dual Role + Third-Party Support)
	bookedByUserID    string                             // The authenticated user who created the order
	assignedCaptainID *string                            // The captain fulfilling the order (nil until assigned)
	sender            *valueobjects.DeliveryContact      // Sender details (may not be a user)
	receiver          *valueobjects.DeliveryContact      // Receiver details (may not be a user)

	// Locations
	pickupAddress   *valueobjects.OrderAddress
	deliveryAddress *valueobjects.OrderAddress

	// Parcel Details (for PARCEL orders)
	parcel *valueobjects.Parcel

	// Pricing
	pricing *valueobjects.PricingBreakdown

	// State Machine
	state          OrderState
	stateHistory   []OrderStateTransition
	stateUpdatedAt time.Time

	// OTPs for Security
	pickupOTP    string
	deliveryOTP  string
	otpExpiresAt time.Time

	// Timing
	scheduledPickupTime *time.Time // For scheduled orders
	actualPickupTime    *time.Time
	estimatedDelivery   *time.Time
	actualDeliveryTime  *time.Time

	// Waiting Time & Charges
	waitingStartTime *time.Time
	waitingMinutes   int
	waitingCharges   *valueobjects.Money

	// Cancellation
	cancellationInfo *valueobjects.CancellationInfo

	// Payment
	paymentMethod   PaymentMethod
	paymentStatus   PaymentStatus
	paymentIntentID *string // For online payments

	// Notes & Instructions
	customerNotes string
	captainNotes  string
	internalNotes string

	// Metadata
	createdAt time.Time
	updatedAt time.Time
	deletedAt *time.Time
}

// OrderType enum
type OrderType string

const (
	OrderTypeParcel    OrderType = "PARCEL"
	OrderTypeRide      OrderType = "RIDE"       // Future
	OrderTypeMultiStop OrderType = "MULTI_STOP" // Future
	OrderTypeScheduled OrderType = "SCHEDULED"  // Future
)

// OrderState represents the order lifecycle state
type OrderState string

const (
	// Initial States
	OrderStateCreated        OrderState = "CREATED"
	OrderStatePriceEstimated OrderState = "PRICE_ESTIMATED"

	// Matching States
	OrderStateSearchingCaptain OrderState = "SEARCHING_CAPTAIN"
	OrderStateCaptainAssigned  OrderState = "CAPTAIN_ASSIGNED"
	OrderStateCaptainAccepted  OrderState = "CAPTAIN_ACCEPTED"

	// Fulfillment States
	OrderStateCaptainEnRoute OrderState = "CAPTAIN_EN_ROUTE" // Captain heading to pickup
	OrderStatePickedUp       OrderState = "PICKED_UP"
	OrderStateInTransit      OrderState = "IN_TRANSIT"
	OrderStateAtDelivery     OrderState = "AT_DELIVERY" // Captain at delivery location
	OrderStateDelivered      OrderState = "DELIVERED"

	// Terminal States
	OrderStateCompleted OrderState = "COMPLETED"
	OrderStateCancelled OrderState = "CANCELLED"
	OrderStateFailed    OrderState = "FAILED"
)

// OrderStateTransition tracks state changes for audit
type OrderStateTransition struct {
	fromState   OrderState
	toState     OrderState
	triggeredBy string // UserID who triggered the change
	reason      string
	timestamp   time.Time
}

// PaymentMethod enum
type PaymentMethod string

const (
	PaymentMethodCash   PaymentMethod = "CASH"
	PaymentMethodWallet PaymentMethod = "WALLET"
	PaymentMethodOnline PaymentMethod = "ONLINE" // Credit card, UPI, etc.
)

// PaymentStatus enum
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusCaptured  PaymentStatus = "CAPTURED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"
	PaymentStatusCancelled PaymentStatus = "CANCELLED"
)

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewOrder creates a new order (CREATED state)
func NewOrder(
	id string,
	orderType OrderType,
	bookedByUserID string,
	sender *valueobjects.DeliveryContact,
	receiver *valueobjects.DeliveryContact,
	pickupAddress *valueobjects.OrderAddress,
	deliveryAddress *valueobjects.OrderAddress,
	parcel *valueobjects.Parcel,
	paymentMethod PaymentMethod,
	customerNotes string,
	scheduledPickupTime *time.Time,
) (*Order, error) {
	// Validation
	if id == "" {
		return nil, errors.New("order ID is required")
	}
	if bookedByUserID == "" {
		return nil, errors.New("booker user ID is required")
	}
	if sender == nil {
		return nil, errors.New("sender details are required")
	}
	if receiver == nil {
		return nil, errors.New("receiver details are required")
	}
	if pickupAddress == nil {
		return nil, errors.New("pickup address is required")
	}
	if deliveryAddress == nil {
		return nil, errors.New("delivery address is required")
	}

	// Parcel validation for PARCEL orders
	if orderType == OrderTypeParcel && parcel == nil {
		return nil, errors.New("parcel details required for parcel orders")
	}

	// Validate payment method
	if !isValidPaymentMethod(paymentMethod) {
		return nil, errors.New("invalid payment method")
	}

	now := time.Now()

	order := &Order{
		id:                  id,
		version:             1,
		orderType:           orderType,
		bookedByUserID:      bookedByUserID,
		sender:              sender,
		receiver:            receiver,
		pickupAddress:       pickupAddress,
		deliveryAddress:     deliveryAddress,
		parcel:              parcel,
		state:               OrderStateCreated,
		stateHistory:        []OrderStateTransition{},
		paymentMethod:       paymentMethod,
		paymentStatus:       PaymentStatusPending,
		customerNotes:       customerNotes,
		scheduledPickupTime: scheduledPickupTime,
		createdAt:           now,
		updatedAt:           now,
		stateUpdatedAt:      now,
	}

	// Record initial state
	order.recordStateTransition(OrderStateCreated, OrderStateCreated, bookedByUserID, "Order created")

	return order, nil
}

// ============================================================================
// STATE MACHINE - Core Domain Logic
// ============================================================================

// validTransitions defines allowed state transitions
var validTransitions = map[OrderState][]OrderState{
	OrderStateCreated: {
		OrderStatePriceEstimated,
		OrderStateCancelled,
		OrderStateFailed,
	},
	OrderStatePriceEstimated: {
		OrderStateSearchingCaptain,
		OrderStateCancelled,
		OrderStateFailed,
	},
	OrderStateSearchingCaptain: {
		OrderStateCaptainAssigned,
		OrderStateCancelled,
		OrderStateFailed,
	},
	OrderStateCaptainAssigned: {
		OrderStateCaptainAccepted,
		OrderStateSearchingCaptain, // Captain declined/timeout
		OrderStateCancelled,
		OrderStateFailed,
	},
	OrderStateCaptainAccepted: {
		OrderStateCaptainEnRoute,
		OrderStateCancelled, // Can still cancel before pickup
	},
	OrderStateCaptainEnRoute: {
		OrderStatePickedUp,
		OrderStateCancelled, // Can still cancel before pickup
	},
	OrderStatePickedUp: {
		OrderStateInTransit,
		OrderStateCancelled, // Admin/system only
	},
	OrderStateInTransit: {
		OrderStateAtDelivery,
		OrderStateCancelled, // Admin/system only
	},
	OrderStateAtDelivery: {
		OrderStateDelivered,
		OrderStateCancelled, // Admin/system only (rare)
	},
	OrderStateDelivered: {
		OrderStateCompleted,
	},
	// Terminal states have no transitions
	OrderStateCompleted: {},
	OrderStateCancelled: {},
	OrderStateFailed:    {},
}

// CanTransitionTo checks if state transition is valid
func (o *Order) CanTransitionTo(newState OrderState) bool {
	allowedStates, exists := validTransitions[o.state]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if allowed == newState {
			return true
		}
	}
	return false
}

// TransitionTo changes order state with validation
func (o *Order) TransitionTo(newState OrderState, triggeredBy string, reason string) error {
	if !o.CanTransitionTo(newState) {
		return fmt.Errorf("invalid state transition from %s to %s", o.state, newState)
	}

	oldState := o.state
	o.state = newState
	o.stateUpdatedAt = time.Now()
	o.updatedAt = time.Now()
	o.version++

	o.recordStateTransition(oldState, newState, triggeredBy, reason)

	return nil
}

// recordStateTransition adds to state history
func (o *Order) recordStateTransition(from, to OrderState, triggeredBy, reason string) {
	o.stateHistory = append(o.stateHistory, OrderStateTransition{
		fromState:   from,
		toState:     to,
		triggeredBy: triggeredBy,
		reason:      reason,
		timestamp:   time.Now(),
	})
}

// ============================================================================
// PRICING
// ============================================================================

// SetPricing sets the pricing breakdown and transitions to PRICE_ESTIMATED
func (o *Order) SetPricing(pricing *valueobjects.PricingBreakdown, triggeredBy string) error {
	if o.state != OrderStateCreated {
		return errors.New("pricing can only be set in CREATED state")
	}

	if pricing == nil {
		return errors.New("pricing cannot be nil")
	}

	o.pricing = pricing
	return o.TransitionTo(OrderStatePriceEstimated, triggeredBy, "Price calculated")
}

// ============================================================================
// CAPTAIN ASSIGNMENT
// ============================================================================

// StartCaptainSearch transitions to SEARCHING_CAPTAIN state
func (o *Order) StartCaptainSearch(triggeredBy string) error {
	if o.state != OrderStatePriceEstimated {
		return errors.New("can only start captain search after price estimation")
	}

	return o.TransitionTo(OrderStateSearchingCaptain, triggeredBy, "Searching for captain")
}

// AssignCaptain assigns a captain to the order
func (o *Order) AssignCaptain(captainUserID string, triggeredBy string) error {
	if o.state != OrderStateSearchingCaptain {
		return errors.New("can only assign captain in SEARCHING_CAPTAIN state")
	}

	// Business Rule: Prevent self-assignment
	if captainUserID == o.bookedByUserID {
		return errors.New("user cannot be both booker and captain for the same order")
	}

	o.assignedCaptainID = &captainUserID
	return o.TransitionTo(OrderStateCaptainAssigned, triggeredBy, "Captain assigned")
}

// CaptainAccept marks captain as accepted
func (o *Order) CaptainAccept(captainUserID string) error {
	if o.state != OrderStateCaptainAssigned {
		return errors.New("invalid state for captain acceptance")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can accept")
	}

	return o.TransitionTo(OrderStateCaptainAccepted, captainUserID, "Captain accepted order")
}

// CaptainDecline handles captain declining order
func (o *Order) CaptainDecline(captainUserID string, reason string) error {
	if o.state != OrderStateCaptainAssigned {
		return errors.New("invalid state for captain decline")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can decline")
	}

	// Clear captain assignment
	o.assignedCaptainID = nil

	// Go back to searching
	return o.TransitionTo(OrderStateSearchingCaptain, captainUserID, "Captain declined: "+reason)
}

// CaptainEnRoute marks captain as heading to pickup
func (o *Order) CaptainEnRoute(captainUserID string) error {
	if o.state != OrderStateCaptainAccepted {
		return errors.New("invalid state for captain en route")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can mark en route")
	}

	return o.TransitionTo(OrderStateCaptainEnRoute, captainUserID, "Captain en route to pickup")
}

// ============================================================================
// PICKUP & DELIVERY
// ============================================================================

// GenerateOTPs generates pickup and delivery OTPs
func (o *Order) GenerateOTPs() error {
	if o.state != OrderStateCaptainAccepted && o.state != OrderStateCaptainEnRoute {
		return errors.New("OTPs can only be generated after captain accepts")
	}

	// Generate 6-digit OTPs
	pickupOTP, err := generateSecureOTP()
	if err != nil {
		return fmt.Errorf("failed to generate pickup OTP: %w", err)
	}

	deliveryOTP, err := generateSecureOTP()
	if err != nil {
		return fmt.Errorf("failed to generate delivery OTP: %w", err)
	}

	o.pickupOTP = pickupOTP
	o.deliveryOTP = deliveryOTP
	o.otpExpiresAt = time.Now().Add(24 * time.Hour)

	return nil
}

// ConfirmPickup verifies OTP and marks order as picked up
func (o *Order) ConfirmPickup(captainUserID string, otp string) error {
	if o.state != OrderStateCaptainEnRoute {
		return errors.New("invalid state for pickup confirmation")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can confirm pickup")
	}

	// Verify OTP
	if o.pickupOTP != otp {
		return errors.New("invalid pickup OTP")
	}

	if time.Now().After(o.otpExpiresAt) {
		return errors.New("pickup OTP has expired")
	}

	now := time.Now()
	o.actualPickupTime = &now

	return o.TransitionTo(OrderStatePickedUp, captainUserID, "Parcel picked up")
}

// StartTransit marks order as in transit
func (o *Order) StartTransit(captainUserID string) error {
	if o.state != OrderStatePickedUp {
		return errors.New("invalid state for starting transit")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can start transit")
	}

	return o.TransitionTo(OrderStateInTransit, captainUserID, "In transit to delivery")
}

// ArriveAtDelivery marks captain as arrived at delivery location
func (o *Order) ArriveAtDelivery(captainUserID string) error {
	if o.state != OrderStateInTransit {
		return errors.New("invalid state for arrival at delivery")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can mark arrival")
	}

	return o.TransitionTo(OrderStateAtDelivery, captainUserID, "Arrived at delivery location")
}

// ConfirmDelivery verifies OTP and marks order as delivered
func (o *Order) ConfirmDelivery(captainUserID string, otp string) error {
	if o.state != OrderStateAtDelivery {
		return errors.New("invalid state for delivery confirmation")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can confirm delivery")
	}

	// Verify OTP
	if o.deliveryOTP != otp {
		return errors.New("invalid delivery OTP")
	}

	if time.Now().After(o.otpExpiresAt) {
		return errors.New("delivery OTP has expired")
	}

	now := time.Now()
	o.actualDeliveryTime = &now

	return o.TransitionTo(OrderStateDelivered, captainUserID, "Parcel delivered")
}

// Complete marks order as completed (after delivery)
func (o *Order) Complete(triggeredBy string) error {
	if o.state != OrderStateDelivered {
		return errors.New("can only complete delivered orders")
	}

	// Final payment capture happens here
	if o.paymentMethod == PaymentMethodOnline && o.paymentStatus == PaymentStatusPending {
		o.paymentStatus = PaymentStatusCaptured
	}

	return o.TransitionTo(OrderStateCompleted, triggeredBy, "Order completed successfully")
}

// ============================================================================
// CANCELLATION
// ============================================================================

// Cancel cancels the order with proper validation
func (o *Order) Cancel(
	cancelledBy string,
	cancelledByRole string, // CUSTOMER, CAPTAIN, ADMIN, SYSTEM
	reason string,
	refundAmount *valueobjects.Money,
	penaltyAmount *valueobjects.Money,
) error {
	// Terminal states cannot be cancelled
	if o.IsTerminalState() {
		return errors.New("cannot cancel order in terminal state")
	}

	// Business Rule: After pickup, only admin can cancel
	if o.state == OrderStatePickedUp || o.state == OrderStateInTransit || o.state == OrderStateAtDelivery {
		if cancelledByRole != "ADMIN" && cancelledByRole != "SYSTEM" {
			return errors.New("only admin can cancel order after pickup")
		}
	}

	// Business Rule: Customer can cancel before pickup
	if cancelledByRole == "CUSTOMER" {
		if cancelledBy != o.bookedByUserID {
			return errors.New("only the booker can cancel this order")
		}
	}

	// Business Rule: Captain can only cancel in specific states
	if cancelledByRole == "CAPTAIN" {
		if o.state != OrderStateCaptainAssigned && o.state != OrderStateCaptainAccepted {
			return errors.New("captain can only cancel before starting trip")
		}
	}

	o.cancellationInfo = &valueobjects.CancellationInfo{
		CancelledBy:     cancelledBy,
		CancelledByRole: cancelledByRole,
		Reason:          reason,
		RefundAmount:    refundAmount,
		PenaltyAmount:   penaltyAmount,
		CancelledAt:     time.Now(),
	}

	// Update payment status
	if refundAmount != nil && !refundAmount.IsZero() {
		o.paymentStatus = PaymentStatusRefunded
	} else {
		o.paymentStatus = PaymentStatusCancelled
	}

	return o.TransitionTo(OrderStateCancelled, cancelledBy, "Order cancelled: "+reason)
}

// ============================================================================
// WAITING TIME & CHARGES
// ============================================================================

// StartWaiting marks the start of waiting time
func (o *Order) StartWaiting(captainUserID string) error {
	if o.state != OrderStateCaptainEnRoute && o.state != OrderStateAtDelivery {
		return errors.New("can only start waiting when captain is at location")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can start waiting timer")
	}

	if o.waitingStartTime != nil {
		return errors.New("waiting already started")
	}

	now := time.Now()
	o.waitingStartTime = &now
	return nil
}

// EndWaiting calculates waiting charges
func (o *Order) EndWaiting(captainUserID string, waitingChargePerMinute *valueobjects.Money) error {
	if o.waitingStartTime == nil {
		return errors.New("waiting not started")
	}

	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can end waiting")
	}

	// Calculate waiting minutes
	elapsed := time.Since(*o.waitingStartTime)
	minutes := int(elapsed.Minutes())

	// Only charge after free waiting period (e.g., 5 minutes)
	freeMinutes := 5
	chargeableMinutes := minutes - freeMinutes
	if chargeableMinutes < 0 {
		chargeableMinutes = 0
	}

	o.waitingMinutes = minutes

	// Calculate charges
	if chargeableMinutes > 0 {
		charges, err := waitingChargePerMinute.Multiply(float64(chargeableMinutes))
		if err != nil {
			return fmt.Errorf("failed to calculate waiting charges: %w", err)
		}
		o.waitingCharges = charges

		// Add to total pricing
		if o.pricing != nil {
			if err := o.pricing.AddWaitingCharges(charges); err != nil {
				return fmt.Errorf("failed to add waiting charges to pricing: %w", err)
			}
		}
	}

	return nil
}

// ============================================================================
// NOTES & METADATA
// ============================================================================

// AddCaptainNotes adds notes from captain
func (o *Order) AddCaptainNotes(captainUserID string, notes string) error {
	if o.assignedCaptainID == nil || *o.assignedCaptainID != captainUserID {
		return errors.New("only assigned captain can add notes")
	}

	o.captainNotes = notes
	o.updatedAt = time.Now()
	return nil
}

// AddInternalNotes adds admin/system notes
func (o *Order) AddInternalNotes(notes string) {
	o.internalNotes = notes
	o.updatedAt = time.Now()
}

// UpdateEstimatedDelivery updates ETA
func (o *Order) UpdateEstimatedDelivery(eta time.Time) {
	o.estimatedDelivery = &eta
	o.updatedAt = time.Now()
}

// SetPaymentIntentID sets the payment intent ID for online payments
func (o *Order) SetPaymentIntentID(intentID string) {
	o.paymentIntentID = &intentID
	o.updatedAt = time.Now()
}

// ============================================================================
// QUERY METHODS
// ============================================================================

// IsTerminalState checks if order is in terminal state
func (o *Order) IsTerminalState() bool {
	return o.state == OrderStateCompleted ||
		o.state == OrderStateCancelled ||
		o.state == OrderStateFailed
}

// IsActive checks if order is active (not terminal)
func (o *Order) IsActive() bool {
	return !o.IsTerminalState()
}

// CanBeCancelledByCustomer checks if customer can cancel
func (o *Order) CanBeCancelledByCustomer() bool {
	// Customer can cancel before pickup
	return o.state == OrderStatePriceEstimated ||
		o.state == OrderStateSearchingCaptain ||
		o.state == OrderStateCaptainAssigned ||
		o.state == OrderStateCaptainAccepted ||
		o.state == OrderStateCaptainEnRoute
}

// CanBeCancelledByCaptain checks if captain can cancel
func (o *Order) CanBeCancelledByCaptain() bool {
	return o.state == OrderStateCaptainAssigned ||
		o.state == OrderStateCaptainAccepted
}

// RequiresPaymentCapture checks if payment needs to be captured
func (o *Order) RequiresPaymentCapture() bool {
	return o.state == OrderStateCompleted &&
		o.paymentMethod == PaymentMethodOnline &&
		o.paymentStatus == PaymentStatusPending
}

// GetDurationMinutes calculates order duration in minutes
func (o *Order) GetDurationMinutes() int {
	if o.actualPickupTime == nil {
		return 0
	}

	endTime := time.Now()
	if o.actualDeliveryTime != nil {
		endTime = *o.actualDeliveryTime
	}

	duration := endTime.Sub(*o.actualPickupTime)
	return int(duration.Minutes())
}

// HasAssignedCaptain checks if order has a captain assigned
func (o *Order) HasAssignedCaptain() bool {
	return o.assignedCaptainID != nil
}

// IsPickedUp checks if parcel has been picked up
func (o *Order) IsPickedUp() bool {
	return o.actualPickupTime != nil
}

// IsDelivered checks if parcel has been delivered
func (o *Order) IsDelivered() bool {
	return o.actualDeliveryTime != nil
}

// ============================================================================
// GETTERS (All fields are private, accessed via methods)
// ============================================================================

func (o *Order) ID() string                                        { return o.id }
func (o *Order) Version() int                                      { return o.version }
func (o *Order) OrderType() OrderType                              { return o.orderType }
func (o *Order) BookedByUserID() string                            { return o.bookedByUserID }
func (o *Order) AssignedCaptainID() *string                        { return o.assignedCaptainID }
func (o *Order) Sender() *valueobjects.DeliveryContact             { return o.sender }
func (o *Order) Receiver() *valueobjects.DeliveryContact           { return o.receiver }
func (o *Order) PickupAddress() *valueobjects.OrderAddress         { return o.pickupAddress }
func (o *Order) DeliveryAddress() *valueobjects.OrderAddress       { return o.deliveryAddress }
func (o *Order) Parcel() *valueobjects.Parcel                      { return o.parcel }
func (o *Order) Pricing() *valueobjects.PricingBreakdown           { return o.pricing }
func (o *Order) State() OrderState                                 { return o.state }
func (o *Order) StateHistory() []OrderStateTransition              { return o.stateHistory }
func (o *Order) StateUpdatedAt() time.Time                         { return o.stateUpdatedAt }
func (o *Order) PickupOTP() string                                 { return o.pickupOTP }
func (o *Order) DeliveryOTP() string                               { return o.deliveryOTP }
func (o *Order) OTPExpiresAt() time.Time                           { return o.otpExpiresAt }
func (o *Order) ScheduledPickupTime() *time.Time                   { return o.scheduledPickupTime }
func (o *Order) ActualPickupTime() *time.Time                      { return o.actualPickupTime }
func (o *Order) EstimatedDelivery() *time.Time                     { return o.estimatedDelivery }
func (o *Order) ActualDeliveryTime() *time.Time                    { return o.actualDeliveryTime }
func (o *Order) WaitingStartTime() *time.Time                      { return o.waitingStartTime }
func (o *Order) WaitingMinutes() int                               { return o.waitingMinutes }
func (o *Order) WaitingCharges() *valueobjects.Money               { return o.waitingCharges }
func (o *Order) CancellationInfo() *valueobjects.CancellationInfo  { return o.cancellationInfo }
func (o *Order) PaymentMethod() PaymentMethod                      { return o.paymentMethod }
func (o *Order) PaymentStatus() PaymentStatus                      { return o.paymentStatus }
func (o *Order) PaymentIntentID() *string                          { return o.paymentIntentID }
func (o *Order) CustomerNotes() string                             { return o.customerNotes }
func (o *Order) CaptainNotes() string                              { return o.captainNotes }
func (o *Order) InternalNotes() string                             { return o.internalNotes }
func (o *Order) CreatedAt() time.Time                              { return o.createdAt }
func (o *Order) UpdatedAt() time.Time                              { return o.updatedAt }
func (o *Order) DeletedAt() *time.Time                             { return o.deletedAt }

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// generateSecureOTP generates a cryptographically secure 6-digit OTP
func generateSecureOTP() (string, error) {
	// Generate a random number between 100000 and 999999
	max := big.NewInt(900000) // 999999 - 100000
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	
	otp := n.Int64() + 100000
	return fmt.Sprintf("%06d", otp), nil
}

// isValidPaymentMethod validates payment method
func isValidPaymentMethod(method PaymentMethod) bool {
	switch method {
	case PaymentMethodCash, PaymentMethodWallet, PaymentMethodOnline:
		return true
	default:
		return false
	}
}

// ============================================================================
// STATE TRANSITION GETTERS
// ============================================================================

func (t OrderStateTransition) FromState() OrderState { return t.fromState }
func (t OrderStateTransition) ToState() OrderState   { return t.toState }
func (t OrderStateTransition) TriggeredBy() string   { return t.triggeredBy }
func (t OrderStateTransition) Reason() string        { return t.reason }
func (t OrderStateTransition) Timestamp() time.Time  { return t.timestamp }