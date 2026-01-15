package models

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	gorm.Model

	// Order Identification
	OrderNumber string `gorm:"uniqueIndex;size:50;not null"` // e.g., "ORD-2024-001234"

	// Customer & Captain
	CustomerID uint  `gorm:"not null;index"`
	CaptainID  *uint `gorm:"index"` // Nullable until captain accepts

	// Addresses
	PickupAddressID   uint `gorm:"not null;index"`
	DeliveryAddressID uint `gorm:"not null;index"`

	// Pickup Details
	PickupName         string  `gorm:"size:100;not null"`
	PickupPhone        string  `gorm:"size:20;not null"`
	PickupLat          float64 `gorm:"type:decimal(10,8);not null"`
	PickupLng          float64 `gorm:"type:decimal(11,8);not null"`
	PickupAddress      string  `gorm:"type:text;not null"`
	PickupInstructions string  `gorm:"type:text"`

	// Delivery Details
	DeliveryName         string  `gorm:"size:100;not null"`
	DeliveryPhone        string  `gorm:"size:20;not null"`
	DeliveryLat          float64 `gorm:"type:decimal(10,8);not null"`
	DeliveryLng          float64 `gorm:"type:decimal(11,8);not null"`
	DeliveryAddress      string  `gorm:"type:text;not null"`
	DeliveryInstructions string  `gorm:"type:text"`

	// Parcel Details
	ParcelType        string  `gorm:"size:50;not null"`  // DOCUMENT / PACKAGE / FOOD / FRAGILE / ELECTRONICS
	ParcelWeight      float64 `gorm:"type:decimal(5,2)"` // in KG
	ParcelDescription string  `gorm:"type:text"`
	ParcelValue       float64 `gorm:"type:decimal(10,2)"` // Declared value for insurance
	ParcelImages      string  `gorm:"type:jsonb"`         // Array of image URLs

	// Waiting & Timing
	DefaultWaitingTimeMinutes int     `gorm:"default:5"`                       // Free waiting time
	ActualWaitingTimeMinutes  int     `gorm:"default:0"`                       // Actual waiting time
	WaitingChargePerMinute    float64 `gorm:"type:decimal(10,2);default:2.00"` // ₹2/min after free time
	TotalWaitingCharge        float64 `gorm:"type:decimal(10,2);default:0"`

	// Arrival & Timing
	EstimatedArrivalTime *time.Time // When captain will arrive at pickup
	ActualArrivalTime    *time.Time // When captain actually arrived
	CaptainArrivedAt     *time.Time // When captain marked as arrived

	// Distance & Time
	EstimatedDistance float64 `gorm:"type:decimal(8,2)"` // in KM
	ActualDistance    float64 `gorm:"type:decimal(8,2)"` // in KM
	EstimatedDuration int     // in minutes
	ActualDuration    int     // in minutes

	// Pricing
	BasePrice       float64 `gorm:"type:decimal(10,2);not null"`
	DistancePrice   float64 `gorm:"type:decimal(10,2);default:0"`
	SurgePrice      float64 `gorm:"type:decimal(10,2);default:0"` // Dynamic pricing
	DiscountAmount  float64 `gorm:"type:decimal(10,2);default:0"`
	TaxAmount       float64 `gorm:"type:decimal(10,2);default:0"`
	TotalPrice      float64 `gorm:"type:decimal(10,2);not null"`
	CaptainEarnings float64 `gorm:"type:decimal(10,2);default:0"`
	PlatformFee     float64 `gorm:"type:decimal(10,2);default:0"`
	Currency        string  `gorm:"default:'INR';size:3"`

	// Cancellation & Refund
	CancellationFee          float64 `gorm:"type:decimal(10,2);default:0"`
	CancellationRefundAmount float64 `gorm:"type:decimal(10,2);default:0"`
	RefundedBy               string  `gorm:"size:50"` // CUSTOMER / CAPTAIN / ADMIN / SYSTEM
	RefundedAt               *time.Time
	RefundTransactionID      string `gorm:"size:100"`

	// Additional charges
	TollCharges             float64 `gorm:"type:decimal(10,2);default:0"`
	ParkingCharges          float64 `gorm:"type:decimal(10,2);default:0"`
	AdditionalCharges       float64 `gorm:"type:decimal(10,2);default:0"`
	AdditionalChargesReason string  `gorm:"type:text"`

	// Coupon & Promo
	CouponCode    string  `gorm:"size:50"`
	PromoDiscount float64 `gorm:"type:decimal(10,2);default:0"`

	// Status & Timeline
	Status string `gorm:"not null;index;default:'PENDING'"`
	// PENDING -> ACCEPTED -> CAPTAIN_ARRIVING -> PICKED_UP -> IN_TRANSIT -> DELIVERED / CANCELLED

	StatusHistory string `gorm:"type:jsonb"` // Array of status changes with timestamps

	// Timestamps for different stages
	PlacedAt        time.Time `gorm:"not null"`
	AcceptedAt      *time.Time
	PickupArrivedAt *time.Time
	PickedUpAt      *time.Time
	DeliveredAt     *time.Time
	CancelledAt     *time.Time

	// Cancellation
	CancellationReason string `gorm:"type:text"`
	CancelledBy        string `gorm:"size:20"` // CUSTOMER / CAPTAIN / SYSTEM

	// Verification
	PickupOTP     string `gorm:"size:6"` // OTP to verify pickup
	DeliveryOTP   string `gorm:"size:6"` // OTP to verify delivery
	IsOTPVerified bool   `gorm:"default:false"`

	// Proof of Delivery
	DeliveryPhoto     string `gorm:"type:text"` // Photo proof URL
	DeliverySignature string `gorm:"type:text"` // Signature data

	// Payment
	PaymentMethod string `gorm:"size:50;not null"`           // CASH / CARD / WALLET / UPI
	PaymentStatus string `gorm:"not null;default:'PENDING'"` // PENDING / PAID / REFUNDED / FAILED
	PaymentID     string `gorm:"size:100"`                   // External payment gateway ID
	PaidAt        *time.Time

	// Rating & Feedback
	CustomerRating   *float64 `gorm:"type:decimal(2,1)"` // Customer rates captain (1-5)
	CaptainRating    *float64 `gorm:"type:decimal(2,1)"` // Captain rates customer (1-5)
	CustomerFeedback string   `gorm:"type:text"`
	CaptainFeedback  string   `gorm:"type:text"`

	// Tracking
	TrackingURL string `gorm:"size:500"` // Real-time tracking URL

	// Insurance & Safety
	IsInsured       bool    `gorm:"default:false"`
	InsuranceAmount float64 `gorm:"type:decimal(10,2);default:0"`

	// Schedule
	IsScheduled       bool `gorm:"default:false"`
	ScheduledPickupAt *time.Time

	// Priority
	IsPriority bool `gorm:"default:false;index"` // Express delivery

	// Relationships
	Customer User  `gorm:"foreignKey:CustomerID"`
	Captain  *User `gorm:"foreignKey:CaptainID"`
	// PickupAddress   Address       `gorm:"foreignKey:PickupAddressID"`
	// DeliveryAddress Address       `gorm:"foreignKey:DeliveryAddressID"`
	Ratings      []Rating      `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	Transactions []Transaction `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name
func (Order) TableName() string {
	return "orders"
}

// IsCompleted checks if order is in final state
func (o *Order) IsCompleted() bool {
	return o.Status == "DELIVERED" || o.Status == "CANCELLED"
}

// CanCancel checks if order can be cancelled
func (o *Order) CanCancel() bool {
	return o.Status == "PENDING" || o.Status == "ACCEPTED"
}

// CalculateTotal calculates total order price
func (o *Order) CalculateTotal() {
	o.TotalPrice = o.BasePrice + o.DistancePrice + o.SurgePrice + o.TaxAmount - o.DiscountAmount - o.PromoDiscount
	if o.TotalPrice < 0 {
		o.TotalPrice = 0
	}
}

// CalculateCaptainEarnings calculates captain's earnings (platform takes commission)
func (o *Order) CalculateCaptainEarnings(platformCommissionPercent float64) {
	totalEarnable := o.BasePrice + o.DistancePrice + o.SurgePrice
	o.PlatformFee = totalEarnable * platformCommissionPercent / 100
	o.CaptainEarnings = totalEarnable - o.PlatformFee
}

// CalculateTotalWithWaiting calculates total including waiting charges
func (o *Order) CalculateTotalWithWaiting() {
	// Calculate waiting charges if exceeded free time
	if o.ActualWaitingTimeMinutes > o.DefaultWaitingTimeMinutes {
		chargeableMinutes := o.ActualWaitingTimeMinutes - o.DefaultWaitingTimeMinutes
		o.TotalWaitingCharge = float64(chargeableMinutes) * o.WaitingChargePerMinute
	}

	// Add all charges
	o.TotalPrice = o.BasePrice + o.DistancePrice + o.SurgePrice +
		o.TaxAmount + o.TotalWaitingCharge +
		o.TollCharges + o.ParkingCharges + o.AdditionalCharges -
		o.DiscountAmount - o.PromoDiscount
}

// CalculateCancellationFee calculates cancellation fee based on order status
func (o *Order) CalculateCancellationFee(cancelledBy string) float64 {
	switch o.Status {
	case "PENDING":
		return 0 // No fee if cancelled before acceptance
	case "ACCEPTED":
		if cancelledBy == "CUSTOMER" {
			return 20 // ₹20 if customer cancels after acceptance
		}
		return 0
	case "CAPTAIN_ARRIVING", "PICKED_UP":
		if cancelledBy == "CUSTOMER" {
			return 50 // ₹50 if customer cancels after captain started
		}
		return 0
	default:
		return 0
	}
}

// CalculateRefundAmount calculates refund amount
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
