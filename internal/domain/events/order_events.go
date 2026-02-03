package events

import (
	"time"
	"turtle/internal/domain/aggregates"
)

// ============================================================================
// ORDER LIFECYCLE EVENTS
// ============================================================================

// OrderCreated is emitted when a new order is created
type OrderCreated struct {
	OrderID          string
	OrderType        aggregates.OrderType
	BookedByUserID   string
	PickupLocation   string
	DeliveryLocation string
	PaymentMethod    aggregates.PaymentMethod
	OccurredAt_      time.Time
}

func (e OrderCreated) EventType() string   { return "OrderCreated" }
func (e OrderCreated) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderCreated) AggregateID() string { return e.OrderID }

// OrderPriceEstimated is emitted when order pricing is calculated
type OrderPriceEstimated struct {
	OrderID            string
	TotalAmount        int64 // in paise
	DistanceKm         float64
	EstimatedMinutes   int
	SurgeMultiplier    float64
	OccurredAt_        time.Time
}

func (e OrderPriceEstimated) EventType() string   { return "OrderPriceEstimated" }
func (e OrderPriceEstimated) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderPriceEstimated) AggregateID() string { return e.OrderID }

// CaptainSearchStarted is emitted when captain matching begins
type CaptainSearchStarted struct {
	OrderID           string
	PickupLatitude    float64
	PickupLongitude   float64
	SearchRadius      float64
	OccurredAt_       time.Time
}

func (e CaptainSearchStarted) EventType() string   { return "CaptainSearchStarted" }
func (e CaptainSearchStarted) OccurredAt() time.Time { return e.OccurredAt_ }
func (e CaptainSearchStarted) AggregateID() string { return e.OrderID }

// CaptainAssigned is emitted when a captain is assigned to an order
type CaptainAssigned struct {
	OrderID         string
	CaptainID       string
	AssignedAt      time.Time
	AttemptNumber   int
	TimeoutAt       time.Time
	OccurredAt_     time.Time
}

func (e CaptainAssigned) EventType() string   { return "CaptainAssigned" }
func (e CaptainAssigned) OccurredAt() time.Time { return e.OccurredAt_ }
func (e CaptainAssigned) AggregateID() string { return e.OrderID }

// CaptainAccepted is emitted when captain accepts the order
type CaptainAccepted struct {
	OrderID     string
	CaptainID   string
	AcceptedAt  time.Time
	OccurredAt_ time.Time
}

func (e CaptainAccepted) EventType() string   { return "CaptainAccepted" }
func (e CaptainAccepted) OccurredAt() time.Time { return e.OccurredAt_ }
func (e CaptainAccepted) AggregateID() string { return e.OrderID }

// CaptainDeclined is emitted when captain declines the order
type CaptainDeclined struct {
	OrderID     string
	CaptainID   string
	Reason      string
	DeclinedAt  time.Time
	OccurredAt_ time.Time
}

func (e CaptainDeclined) EventType() string   { return "CaptainDeclined" }
func (e CaptainDeclined) OccurredAt() time.Time { return e.OccurredAt_ }
func (e CaptainDeclined) AggregateID() string { return e.OrderID }

// CaptainEnRoute is emitted when captain starts heading to pickup
type CaptainEnRoute struct {
	OrderID              string
	CaptainID            string
	CaptainLatitude      float64
	CaptainLongitude     float64
	EstimatedArrivalTime time.Time
	OccurredAt_          time.Time
}

func (e CaptainEnRoute) EventType() string   { return "CaptainEnRoute" }
func (e CaptainEnRoute) OccurredAt() time.Time { return e.OccurredAt_ }
func (e CaptainEnRoute) AggregateID() string { return e.OrderID }

// OrderPickedUp is emitted when parcel is picked up
type OrderPickedUp struct {
	OrderID     string
	CaptainID   string
	PickupTime  time.Time
	PickupOTP   string // For verification
	OccurredAt_ time.Time
}

func (e OrderPickedUp) EventType() string   { return "OrderPickedUp" }
func (e OrderPickedUp) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderPickedUp) AggregateID() string { return e.OrderID }

// OrderInTransit is emitted when captain starts journey to delivery
type OrderInTransit struct {
	OrderID              string
	CaptainID            string
	EstimatedDeliveryTime time.Time
	DistanceRemaining    float64
	OccurredAt_          time.Time
}

func (e OrderInTransit) EventType() string   { return "OrderInTransit" }
func (e OrderInTransit) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderInTransit) AggregateID() string { return e.OrderID }

// CaptainArrivedAtDelivery is emitted when captain reaches delivery location
type CaptainArrivedAtDelivery struct {
	OrderID            string
	CaptainID          string
	DeliveryLatitude   float64
	DeliveryLongitude  float64
	ArrivedAt          time.Time
	OccurredAt_        time.Time
}

func (e CaptainArrivedAtDelivery) EventType() string   { return "CaptainArrivedAtDelivery" }
func (e CaptainArrivedAtDelivery) OccurredAt() time.Time { return e.OccurredAt_ }
func (e CaptainArrivedAtDelivery) AggregateID() string { return e.OrderID }

// OrderDelivered is emitted when parcel is delivered
type OrderDelivered struct {
	OrderID      string
	CaptainID    string
	DeliveryTime time.Time
	DeliveryOTP  string // For verification
	OccurredAt_  time.Time
}

func (e OrderDelivered) EventType() string   { return "OrderDelivered" }
func (e OrderDelivered) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderDelivered) AggregateID() string { return e.OrderID }

// OrderCompleted is emitted when order is marked as completed
type OrderCompleted struct {
	OrderID         string
	CaptainID       string
	TotalAmount     int64 // in paise
	CaptainEarning  int64 // in paise
	DurationMinutes int
	DistanceKm      float64
	OccurredAt_     time.Time
}

func (e OrderCompleted) EventType() string   { return "OrderCompleted" }
func (e OrderCompleted) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderCompleted) AggregateID() string { return e.OrderID }

// ============================================================================
// CANCELLATION EVENTS
// ============================================================================

// OrderCancelled is emitted when order is cancelled
type OrderCancelled struct {
	OrderID         string
	CancelledBy     string
	CancelledByRole string
	Reason          string
	RefundAmount    int64 // in paise
	PenaltyAmount   int64 // in paise
	PenaltyPoints   int
	OccurredAt_     time.Time
}

func (e OrderCancelled) EventType() string   { return "OrderCancelled" }
func (e OrderCancelled) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderCancelled) AggregateID() string { return e.OrderID }

// OrderFailed is emitted when order fails (no captain, timeout, etc.)
type OrderFailed struct {
	OrderID     string
	Reason      string
	FailureType string // NO_CAPTAIN, TIMEOUT, SYSTEM_ERROR
	OccurredAt_ time.Time
}

func (e OrderFailed) EventType() string   { return "OrderFailed" }
func (e OrderFailed) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderFailed) AggregateID() string { return e.OrderID }

// ============================================================================
// PAYMENT EVENTS
// ============================================================================

// PaymentCaptured is emitted when payment is successfully captured
type PaymentCaptured struct {
	OrderID         string
	TransactionID   string
	Amount          int64 // in paise
	PaymentMethod   string
	PaymentIntentID string
	OccurredAt_     time.Time
}

func (e PaymentCaptured) EventType() string   { return "PaymentCaptured" }
func (e PaymentCaptured) OccurredAt() time.Time { return e.OccurredAt_ }
func (e PaymentCaptured) AggregateID() string { return e.OrderID }

// PaymentFailed is emitted when payment fails
type PaymentFailed struct {
	OrderID         string
	Amount          int64 // in paise
	PaymentMethod   string
	PaymentIntentID string
	FailureReason   string
	OccurredAt_     time.Time
}

func (e PaymentFailed) EventType() string   { return "PaymentFailed" }
func (e PaymentFailed) OccurredAt() time.Time { return e.OccurredAt_ }
func (e PaymentFailed) AggregateID() string { return e.OrderID }

// RefundProcessed is emitted when refund is processed
type RefundProcessed struct {
	OrderID       string
	TransactionID string
	Amount        int64 // in paise
	Reason        string
	OccurredAt_   time.Time
}

func (e RefundProcessed) EventType() string   { return "RefundProcessed" }
func (e RefundProcessed) OccurredAt() time.Time { return e.OccurredAt_ }
func (e RefundProcessed) AggregateID() string { return e.OrderID }

// ============================================================================
// TRACKING EVENTS
// ============================================================================

// CaptainLocationUpdated is emitted when captain's location updates
type CaptainLocationUpdated struct {
	OrderID     string
	CaptainID   string
	Latitude    float64
	Longitude   float64
	Heading     float64 // Direction in degrees
	Speed       float64 // km/h
	OccurredAt_ time.Time
}

func (e CaptainLocationUpdated) EventType() string   { return "CaptainLocationUpdated" }
func (e CaptainLocationUpdated) OccurredAt() time.Time { return e.OccurredAt_ }
func (e CaptainLocationUpdated) AggregateID() string { return e.OrderID }

// ETAUpdated is emitted when estimated arrival time changes
type ETAUpdated struct {
	OrderID           string
	NewETA            time.Time
	PreviousETA       time.Time
	DistanceRemaining float64
	OccurredAt_       time.Time
}

func (e ETAUpdated) EventType() string   { return "ETAUpdated" }
func (e ETAUpdated) OccurredAt() time.Time { return e.OccurredAt_ }
func (e ETAUpdated) AggregateID() string { return e.OrderID }

// ============================================================================
// WAITING TIME EVENTS
// ============================================================================

// WaitingStarted is emitted when waiting timer starts
type WaitingStarted struct {
	OrderID     string
	CaptainID   string
	Location    string // pickup or delivery
	StartedAt   time.Time
	OccurredAt_ time.Time
}

func (e WaitingStarted) EventType() string   { return "WaitingStarted" }
func (e WaitingStarted) OccurredAt() time.Time { return e.OccurredAt_ }
func (e WaitingStarted) AggregateID() string { return e.OrderID }

// WaitingEnded is emitted when waiting timer ends
type WaitingEnded struct {
	OrderID         string
	CaptainID       string
	WaitingMinutes  int
	WaitingCharges  int64 // in paise
	OccurredAt_     time.Time
}

func (e WaitingEnded) EventType() string   { return "WaitingEnded" }
func (e WaitingEnded) OccurredAt() time.Time { return e.OccurredAt_ }
func (e WaitingEnded) AggregateID() string { return e.OrderID }

// ============================================================================
// PENALTY & REWARD EVENTS
// ============================================================================

// PenaltyPointsAdded is emitted when penalty points are added to a user
type PenaltyPointsAdded struct {
	UserID      string
	Points      int
	Reason      string
	TotalPoints int
	OccurredAt_ time.Time
}

func (e PenaltyPointsAdded) EventType() string   { return "PenaltyPointsAdded" }
func (e PenaltyPointsAdded) OccurredAt() time.Time { return e.OccurredAt_ }
func (e PenaltyPointsAdded) AggregateID() string { return e.UserID }

// RewardPointsAdded is emitted when reward points are added to a user
type RewardPointsAdded struct {
	UserID      string
	Points      int
	Reason      string
	TotalPoints int
	OccurredAt_ time.Time
}

func (e RewardPointsAdded) EventType() string   { return "RewardPointsAdded" }
func (e RewardPointsAdded) OccurredAt() time.Time { return e.OccurredAt_ }
func (e RewardPointsAdded) AggregateID() string { return e.UserID }

// UserSuspended is emitted when a user is suspended
type UserSuspended struct {
	UserID          string
	Reason          string
	SuspensionUntil time.Time
	PenaltyPoints   int
	OccurredAt_     time.Time
}

func (e UserSuspended) EventType() string   { return "UserSuspended" }
func (e UserSuspended) OccurredAt() time.Time { return e.OccurredAt_ }
func (e UserSuspended) AggregateID() string { return e.UserID }

// UserUnsuspended is emitted when a user suspension is lifted
type UserUnsuspended struct {
	UserID      string
	Reason      string
	OccurredAt_ time.Time
}

func (e UserUnsuspended) EventType() string   { return "UserUnsuspended" }
func (e UserUnsuspended) OccurredAt() time.Time { return e.OccurredAt_ }
func (e UserUnsuspended) AggregateID() string { return e.UserID }

// ============================================================================
// NOTIFICATION EVENTS
// ============================================================================

// NotificationSent is emitted when a notification is sent to a user
type NotificationSent struct {
	UserID          string
	NotificationType string
	Title           string
	Message         string
	OrderID         *string
	SentVia         string // PUSH, SMS, EMAIL
	OccurredAt_     time.Time
}

func (e NotificationSent) EventType() string   { return "NotificationSent" }
func (e NotificationSent) OccurredAt() time.Time { return e.OccurredAt_ }
func (e NotificationSent) AggregateID() string { return e.UserID }

// ============================================================================
// RATING EVENTS
// ============================================================================

// OrderRated is emitted when an order is rated
type OrderRated struct {
	OrderID       string
	RatedBy       string
	RatedUser     string
	RatedUserRole string // CUSTOMER or CAPTAIN
	Score         float64
	Review        string
	OccurredAt_   time.Time
}

func (e OrderRated) EventType() string   { return "OrderRated" }
func (e OrderRated) OccurredAt() time.Time { return e.OccurredAt_ }
func (e OrderRated) AggregateID() string { return e.OrderID }

// ============================================================================
// ADMIN EVENTS
// ============================================================================

// AdminActionPerformed is emitted when admin performs an action
type AdminActionPerformed struct {
	AdminID     string
	ActionType  string // CANCEL_ORDER, RESET_PENALTY, ADJUST_POINTS, SUSPEND_USER
	TargetType  string // ORDER, USER
	TargetID    string
	Reason      string
	Metadata    map[string]interface{}
	OccurredAt_ time.Time
}

func (e AdminActionPerformed) EventType() string   { return "AdminActionPerformed" }
func (e AdminActionPerformed) OccurredAt() time.Time { return e.OccurredAt_ }
func (e AdminActionPerformed) AggregateID() string { return e.TargetID }
