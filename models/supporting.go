package models

import (
	"time"

	"gorm.io/gorm"
)

// Notification stores push notifications and in-app messages
type Notification struct {
	gorm.Model

	UserID uint `gorm:"not null;index"`

	Title   string `gorm:"size:255;not null"`
	Message string `gorm:"type:text;not null"`
	Type    string `gorm:"size:50;not null;index"` // ORDER_UPDATE / PROMOTION / PAYMENT / SYSTEM

	// Related entities
	OrderID       *uint `gorm:"index"`
	TransactionID *uint `gorm:"index"`

	// Status
	IsRead bool `gorm:"default:false;index"`
	ReadAt *time.Time
	IsSent bool `gorm:"default:false"`
	SentAt *time.Time

	// Deep linking
	ActionURL string `gorm:"size:500"` // URL to open when notification is clicked

	// Metadata
	Data string `gorm:"type:jsonb"` // Additional data as JSON

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name
func (Notification) TableName() string {
	return "notifications"
}

// Coupon stores promotional coupons
type Coupon struct {
	gorm.Model

	Code        string  `gorm:"uniqueIndex;size:50;not null"`
	Description string  `gorm:"type:text"`
	Type        string  `gorm:"size:20;not null"`            // PERCENTAGE / FIXED_AMOUNT / FREE_DELIVERY
	Value       float64 `gorm:"type:decimal(10,2);not null"` // % or amount

	// Limits
	MinOrderValue    float64 `gorm:"type:decimal(10,2);default:0"`
	MaxDiscountValue float64 `gorm:"type:decimal(10,2);default:0"`
	UsageLimit       int     `gorm:"default:0"` // 0 = unlimited
	UsageCount       int     `gorm:"default:0"`
	PerUserLimit     int     `gorm:"default:1"`

	// Validity
	ValidFrom  time.Time
	ValidUntil time.Time
	IsActive   bool `gorm:"default:true;index"`

	// User restrictions
	IsPublic       bool   `gorm:"default:true"` // If false, only specific users can use
	AllowedUserIDs string `gorm:"type:jsonb"`   // Array of user IDs if not public
	FirstOrderOnly bool   `gorm:"default:false"`
}

// TableName specifies the table name
func (Coupon) TableName() string {
	return "coupons"
}

// IsValid checks if coupon is currently valid
func (c *Coupon) IsValid() bool {
	now := time.Now()
	return c.IsActive &&
		now.After(c.ValidFrom) &&
		now.Before(c.ValidUntil) &&
		(c.UsageLimit == 0 || c.UsageCount < c.UsageLimit)
}

// SupportTicket for customer support
type SupportTicket struct {
	gorm.Model

	UserID  uint  `gorm:"not null;index"`
	OrderID *uint `gorm:"index"`

	TicketNumber string `gorm:"uniqueIndex;size:50;not null"` // e.g., "TKT-2024-001234"
	Subject      string `gorm:"size:255;not null"`
	Description  string `gorm:"type:text;not null"`
	Category     string `gorm:"size:50;not null;index"` // ORDER_ISSUE / PAYMENT / ACCOUNT / CAPTAIN_BEHAVIOR / OTHER

	Status   string `gorm:"not null;index;default:'OPEN'"` // OPEN / IN_PROGRESS / RESOLVED / CLOSED
	Priority string `gorm:"size:20;default:'MEDIUM'"`      // LOW / MEDIUM / HIGH / URGENT

	// Assignment
	AssignedToAdminID *uint
	AssignedAt        *time.Time
	ResolvedAt        *time.Time

	// Attachments
	Attachments string `gorm:"type:jsonb"` // Array of file URLs

	// Relationships
	User  User   `gorm:"foreignKey:UserID"`
	Order *Order `gorm:"foreignKey:OrderID"`
	Admin *User  `gorm:"foreignKey:AssignedToAdminID"`
}

// TableName specifies the table name
func (SupportTicket) TableName() string {
	return "support_tickets"
}

// CaptainEarnings tracks captain's daily/weekly earnings summary
type CaptainEarnings struct {
	gorm.Model

	CaptainID uint      `gorm:"not null;index"`
	Date      time.Time `gorm:"not null;index;type:date"`

	// Earnings
	TotalEarnings float64 `gorm:"type:decimal(10,2);default:0"`
	PlatformFee   float64 `gorm:"type:decimal(10,2);default:0"`
	NetEarnings   float64 `gorm:"type:decimal(10,2);default:0"`
	TipsReceived  float64 `gorm:"type:decimal(10,2);default:0"`
	BonusEarnings float64 `gorm:"type:decimal(10,2);default:0"`

	// Statistics
	TotalOrders     int     `gorm:"default:0"`
	CompletedOrders int     `gorm:"default:0"`
	CancelledOrders int     `gorm:"default:0"`
	TotalDistance   float64 `gorm:"type:decimal(10,2);default:0"` // in KM
	OnlineHours     float64 `gorm:"type:decimal(5,2);default:0"`
	AverageRating   float64 `gorm:"type:decimal(3,2);default:0"`

	// Relationships
	Captain User `gorm:"foreignKey:CaptainID"`
}

// TableName specifies the table name
func (CaptainEarnings) TableName() string {
	return "captain_earnings"
}
