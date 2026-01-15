package models

import (
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	gorm.Model

	UserID  uint  `gorm:"not null;index"`
	OrderID *uint `gorm:"index"` // Nullable for wallet transactions

	// Transaction Details
	TransactionID string  `gorm:"uniqueIndex;size:100;not null"` // Unique transaction ID
	Type          string  `gorm:"not null;index"`                // ORDER_PAYMENT / REFUND / WALLET_TOPUP / WITHDRAWAL / COMMISSION
	Amount        float64 `gorm:"type:decimal(10,2);not null"`
	Currency      string  `gorm:"default:'INR';size:3"`

	// Balance tracking
	BalanceBefore float64 `gorm:"type:decimal(10,2)"`
	BalanceAfter  float64 `gorm:"type:decimal(10,2)"`

	// Payment Gateway Details
	PaymentMethod    string `gorm:"size:50"` // CASH / CARD / UPI / WALLET / NET_BANKING
	PaymentGateway   string `gorm:"size:50"` // RAZORPAY / STRIPE / PAYTM
	GatewayOrderID   string `gorm:"size:100"`
	GatewayPaymentID string `gorm:"size:100"`

	// Status
	Status string `gorm:"not null;index;default:'PENDING'"` // PENDING / SUCCESS / FAILED / REFUNDED

	// Metadata
	Description string `gorm:"type:text"`
	Metadata    string `gorm:"type:jsonb"` // Additional data as JSON

	// Timeline
	ProcessedAt   *time.Time
	FailedAt      *time.Time
	FailureReason string `gorm:"type:text"`

	// Relationships
	User  User   `gorm:"foreignKey:UserID"`
	Order *Order `gorm:"foreignKey:OrderID"`
}

// TableName specifies the table name
func (Transaction) TableName() string {
	return "transactions"
}

// IsSuccess checks if transaction was successful
func (t *Transaction) IsSuccess() bool {
	return t.Status == "SUCCESS"
}
