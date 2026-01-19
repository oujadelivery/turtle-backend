package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Order model with critical improvements
type Order struct {
	gorm.Model

	// CRITICAL: Add idempotency key to prevent duplicate orders
	IdempotencyKey string `gorm:"uniqueIndex;size:100"` // Prevent duplicate submissions

	// CRITICAL: Add version for optimistic locking
	Version int `gorm:"not null;default:1;index"` // Optimistic locking

	// Order Identification
	OrderNumber string `gorm:"uniqueIndex;size:50;not null;index"`

	// Customer & Captain
	CustomerID uint  `gorm:"not null;index:idx_customer_status"`
	CaptainID  *uint `gorm:"index:idx_captain_status"`

	// Composite indexes for common queries
	// index:idx_customer_status covers (customer_id, status)
	// index:idx_captain_status covers (captain_id, status)

	// Addresses
	PickupAddressID   uint `gorm:"not null;index"`
	DeliveryAddressID uint `gorm:"not null;index"`

	// Pickup Details
	PickupName         string  `gorm:"size:100;not null"`
	PickupPhone        string  `gorm:"size:20;not null;index"` // Index for phone lookups
	PickupLat          float64 `gorm:"type:decimal(10,8);not null;index:idx_pickup_location"`
	PickupLng          float64 `gorm:"type:decimal(11,8);not null;index:idx_pickup_location"`
	PickupAddress      string  `gorm:"type:text;not null"`
	PickupInstructions string  `gorm:"type:text"`

	// Delivery Details
	DeliveryName         string  `gorm:"size:100;not null"`
	DeliveryPhone        string  `gorm:"size:20;not null;index"`
	DeliveryLat          float64 `gorm:"type:decimal(10,8);not null;index:idx_delivery_location"`
	DeliveryLng          float64 `gorm:"type:decimal(11,8);not null;index:idx_delivery_location"`
	DeliveryAddress      string  `gorm:"type:text;not null"`
	DeliveryInstructions string  `gorm:"type:text"`

	// Parcel Details
	ParcelType        string  `gorm:"size:50;not null;index"` // Index for filtering
	ParcelWeight      float64 `gorm:"type:decimal(5,2)"`
	ParcelDescription string  `gorm:"type:text"`
	ParcelValue       float64 `gorm:"type:decimal(10,2)"`
	ParcelImages      string  `gorm:"type:jsonb"`

	// IMPROVEMENT: Add waiting time tracking
	DefaultWaitingTimeMinutes int     `gorm:"default:5"`
	ActualWaitingTimeMinutes  int     `gorm:"default:0"`
	WaitingChargePerMinute    float64 `gorm:"type:decimal(10,2);default:2.00"`
	TotalWaitingCharge        float64 `gorm:"type:decimal(10,2);default:0"`

	// IMPROVEMENT: Arrival tracking
	EstimatedArrivalTime *time.Time
	ActualArrivalTime    *time.Time
	CaptainArrivedAt     *time.Time

	// Distance & Time
	EstimatedDistance float64 `gorm:"type:decimal(8,2);index"` // Index for analytics
	ActualDistance    float64 `gorm:"type:decimal(8,2)"`
	EstimatedDuration int
	ActualDuration    int

	// Pricing
	BasePrice       float64 `gorm:"type:decimal(10,2);not null"`
	DistancePrice   float64 `gorm:"type:decimal(10,2);default:0"`
	SurgePrice      float64 `gorm:"type:decimal(10,2);default:0"`
	DiscountAmount  float64 `gorm:"type:decimal(10,2);default:0"`
	TaxAmount       float64 `gorm:"type:decimal(10,2);default:0"`
	TotalPrice      float64 `gorm:"type:decimal(10,2);not null;index"` // Index for revenue analytics
	CaptainEarnings float64 `gorm:"type:decimal(10,2);default:0"`
	PlatformFee     float64 `gorm:"type:decimal(10,2);default:0"`
	Currency        string  `gorm:"default:'INR';size:3"`

	// IMPROVEMENT: Enhanced cancellation tracking
	CancellationFee          float64 `gorm:"type:decimal(10,2);default:0"`
	CancellationRefundAmount float64 `gorm:"type:decimal(10,2);default:0"`
	RefundedBy               string  `gorm:"size:50"`
	RefundedAt               *time.Time
	RefundTransactionID      string `gorm:"size:100"`

	// IMPROVEMENT: Additional charges
	TollCharges             float64 `gorm:"type:decimal(10,2);default:0"`
	ParkingCharges          float64 `gorm:"type:decimal(10,2);default:0"`
	AdditionalCharges       float64 `gorm:"type:decimal(10,2);default:0"`
	AdditionalChargesReason string  `gorm:"type:text"`

	// Coupon & Promo
	CouponCode    string  `gorm:"size:50;index"` // Index for coupon analytics
	PromoDiscount float64 `gorm:"type:decimal(10,2);default:0"`

	// Status & Timeline - CRITICAL INDEX
	Status string `gorm:"not null;index:idx_status_created;default:'PENDING'"`
	// Composite index: idx_status_created on (status, created_at) for dashboard queries

	StatusHistory string `gorm:"type:jsonb"`

	// Timestamps for different stages - All indexed for analytics
	PlacedAt        time.Time  `gorm:"not null;index:idx_placed_at"`
	AcceptedAt      *time.Time `gorm:"index"`
	PickupArrivedAt *time.Time
	PickedUpAt      *time.Time `gorm:"index"`
	DeliveredAt     *time.Time `gorm:"index:idx_delivered_at"`
	CancelledAt     *time.Time `gorm:"index"`

	// Cancellation
	CancellationReason string `gorm:"type:text"`
	CancelledBy        string `gorm:"size:20;index"` // Index for analytics

	// Verification
	PickupOTP     string `gorm:"size:10"` // Increased size for alphanumeric OTP
	DeliveryOTP   string `gorm:"size:10"`
	IsOTPVerified bool   `gorm:"default:false"`

	// IMPROVEMENT: Add OTP expiry timestamps
	PickupOTPExpiresAt   *time.Time
	DeliveryOTPExpiresAt *time.Time
	OTPAttempts          int `gorm:"default:0"` // Track failed attempts

	// Proof of Delivery
	DeliveryPhoto     string `gorm:"type:text"`
	DeliverySignature string `gorm:"type:text"`

	// Payment
	PaymentMethod string     `gorm:"size:50;not null;index"` // Index for payment analytics
	PaymentStatus string     `gorm:"not null;default:'PENDING';index"`
	PaymentID     string     `gorm:"size:100;index"` // Index for payment gateway lookups
	PaidAt        *time.Time `gorm:"index"`

	// Rating & Feedback
	CustomerRating   *float64 `gorm:"type:decimal(2,1)"`
	CaptainRating    *float64 `gorm:"type:decimal(2,1)"`
	CustomerFeedback string   `gorm:"type:text"`
	CaptainFeedback  string   `gorm:"type:text"`

	// Tracking
	TrackingURL string `gorm:"size:500"`

	// Insurance & Safety
	IsInsured       bool    `gorm:"default:false;index"` // Index for insurance reports
	InsuranceAmount float64 `gorm:"type:decimal(10,2);default:0"`

	// Schedule
	IsScheduled       bool       `gorm:"default:false;index"`
	ScheduledPickupAt *time.Time `gorm:"index"` // Index for scheduled order queries

	// Priority
	IsPriority bool `gorm:"default:false;index"` // Index for priority filtering

	// IMPROVEMENT: Add fraud detection fields
	RiskScore      float64 `gorm:"type:decimal(3,2);default:0"` // 0-1 risk score
	IsFraudulent   bool    `gorm:"default:false;index"`
	FraudReason    string  `gorm:"type:text"`
	
	// IMPROVEMENT: Add timezone for proper time handling
	Timezone string `gorm:"size:50;default:'Asia/Kolkata'"`

	// Relationships (use eager loading with DataLoader)
	Customer        User          `gorm:"foreignKey:CustomerID"`
	Captain         *User         `gorm:"foreignKey:CaptainID"`
	PickupAddresses   Address       `gorm:"foreignKey:PickupAddressID"`
	DeliveryAddresses Address       `gorm:"foreignKey:DeliveryAddressID"`
	Ratings         []Rating      `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	Transactions    []Transaction `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name
func (Order) TableName() string {
	return "orders"
}

// BeforeCreate hook for validation
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	// Ensure idempotency key exists
	if o.IdempotencyKey == "" {
		return errors.New("idempotency key is required")
	}
	
	// Validate version
	if o.Version == 0 {
		o.Version = 1
	}
	
	// Set OTP expiry (15 minutes from now)
	expiryTime := time.Now().Add(15 * time.Minute)
	o.PickupOTPExpiresAt = &expiryTime
	o.DeliveryOTPExpiresAt = &expiryTime
	
	return nil
}

// Existing methods...
func (o *Order) IsCompleted() bool {
	return o.Status == "DELIVERED" || o.Status == "CANCELLED"
}

func (o *Order) CanCancel() bool {
	return o.Status == "PENDING" || o.Status == "ACCEPTED"
}

func (o *Order) CalculateTotal() {
	o.TotalPrice = o.BasePrice + o.DistancePrice + o.SurgePrice + 
		o.TaxAmount + o.TotalWaitingCharge + o.TollCharges + 
		o.ParkingCharges + o.AdditionalCharges - 
		o.DiscountAmount - o.PromoDiscount
	
	if o.TotalPrice < 0 {
		o.TotalPrice = 0
	}
}

func (o *Order) CalculateCaptainEarnings(platformCommissionPercent float64) {
	totalEarnable := o.BasePrice + o.DistancePrice + o.SurgePrice + 
		o.TollCharges + o.ParkingCharges + o.AdditionalCharges
	o.PlatformFee = totalEarnable * platformCommissionPercent / 100
	o.CaptainEarnings = totalEarnable - o.PlatformFee
}

func (o *Order) CalculateCancellationFee(cancelledBy string) float64 {
	switch o.Status {
	case "PENDING":
		return 0
	case "ACCEPTED":
		if cancelledBy == "CUSTOMER" {
			return 20
		}
		return 0
	case "CAPTAIN_ARRIVING", "PICKED_UP":
		if cancelledBy == "CUSTOMER" {
			return 50
		}
		return 0
	default:
		return 0
	}
}

func (o *Order) CalculateRefundAmount() float64 {
	if o.PaymentStatus != "PAID" {
		return 0
	}
	refund := o.TotalPrice - o.CancellationFee
	if refund < 0 {
		refund = 0
	}
	return refund
}

// IMPROVEMENT: Add method to validate OTP with expiry check
func (o *Order) ValidatePickupOTP(otp string) error {
	if o.OTPAttempts >= 5 {
		return errors.New("maximum OTP attempts exceeded")
	}
	
	if o.PickupOTPExpiresAt != nil && time.Now().After(*o.PickupOTPExpiresAt) {
		return errors.New("OTP has expired")
	}
	
	if o.PickupOTP != otp {
		o.OTPAttempts++
		return errors.New("invalid OTP")
	}
	
	return nil
}

func (o *Order) ValidateDeliveryOTP(otp string) error {
	if o.OTPAttempts >= 5 {
		return errors.New("maximum OTP attempts exceeded")
	}
	
	if o.DeliveryOTPExpiresAt != nil && time.Now().After(*o.DeliveryOTPExpiresAt) {
		return errors.New("OTP has expired")
	}
	
	if o.DeliveryOTP != otp {
		o.OTPAttempts++
		return errors.New("invalid OTP")
	}
	
	return nil
}

