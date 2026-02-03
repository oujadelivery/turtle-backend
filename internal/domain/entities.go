package domain

import "time"

// CancellationEvent represents a cancellation event
type CancellationEvent struct {
	ID                       int64
	OrderID                  string
	UserID                   string
	CancelledByRole          string
	OrderStateAtCancellation string
	Reason                   string
	RefundAmount             int64 // in paise
	PenaltyAmount            int64 // in paise
	PenaltyPoints            int
	CreatedAt                time.Time
}

// UserPenalty tracks user penalty and reward points
type UserPenalty struct {
	ID                 int64
	UserID             string
	TotalPenaltyPoints int
	TotalRewardPoints  int
	NetScore           int
	IsSuspended        bool
	SuspensionUntil    *time.Time
	LastPenaltyAt      *time.Time
	LastRewardAt       *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Transaction represents a payment transaction
type Transaction struct {
	ID              int64
	UserID          string
	OrderID         *string
	Type            string // CHARGE, REFUND, WALLET_TOPUP, WALLET_WITHDRAWAL, CAPTAIN_PAYOUT
	Amount          int64  // in paise
	Currency        string
	Status          string // PENDING, COMPLETED, FAILED, REFUNDED
	PaymentMethod   string
	PaymentIntentID *string
	Description     string
	Metadata        map[string]string // Additional key-value data
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
