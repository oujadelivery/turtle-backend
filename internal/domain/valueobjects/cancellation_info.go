package valueobjects

import (
	"time"
)

// CancellationInfo tracks order cancellation details
// Immutable value object
type CancellationInfo struct {
	CancelledBy     string
	CancelledByRole string // CUSTOMER, CAPTAIN, ADMIN, SYSTEM
	Reason          string
	RefundAmount    *Money
	PenaltyAmount   *Money
	CancelledAt     time.Time
}

// NewCancellationInfo creates a new cancellation info
func NewCancellationInfo(
	cancelledBy string,
	cancelledByRole string,
	reason string,
	refundAmount *Money,
	penaltyAmount *Money,
) *CancellationInfo {
	return &CancellationInfo{
		CancelledBy:     cancelledBy,
		CancelledByRole: cancelledByRole,
		Reason:          reason,
		RefundAmount:    refundAmount,
		PenaltyAmount:   penaltyAmount,
		CancelledAt:     time.Now(),
	}
}

// HasRefund checks if refund was issued
func (ci CancellationInfo) HasRefund() bool {
	return ci.RefundAmount != nil && !ci.RefundAmount.IsZero()
}

// HasPenalty checks if penalty was applied
func (ci CancellationInfo) HasPenalty() bool {
	return ci.PenaltyAmount != nil && !ci.PenaltyAmount.IsZero()
}

// IsCancelledByCustomer checks if cancelled by customer
func (ci CancellationInfo) IsCancelledByCustomer() bool {
	return ci.CancelledByRole == "CUSTOMER"
}

// IsCancelledByCaptain checks if cancelled by captain
func (ci CancellationInfo) IsCancelledByCaptain() bool {
	return ci.CancelledByRole == "CAPTAIN"
}

// IsCancelledByAdmin checks if cancelled by admin
func (ci CancellationInfo) IsCancelledByAdmin() bool {
	return ci.CancelledByRole == "ADMIN"
}

// IsCancelledBySystem checks if cancelled by system
func (ci CancellationInfo) IsCancelledBySystem() bool {
	return ci.CancelledByRole == "SYSTEM"
}