package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
    gorm.Model

    // Basic Info
    FirstName string `gorm:"size:100"`
    LastName  string `gorm:"size:100"`
    ProfilePic string `gorm:"size:500"` // URL to profile image

    // Contact
    Email         *string `gorm:"uniqueIndex;size:255"`
    EmailVerified bool   `gorm:"default:false"`
    Phone         *string `gorm:"uniqueIndex;size:20"`
    PhoneVerified bool   `gorm:"default:false"`

    // Role & Status
    Role   string `gorm:"not null;default:'CUSTOMER';index"` // CUSTOMER / CAPTAIN / ADMIN
    Status string `gorm:"default:'ACTIVE';index"` // ACTIVE / BLOCKED / PENDING_KYC / SUSPENDED

    // Social Login
    Provider   string `gorm:"size:50;index"`
    ProviderID *string `gorm:"uniqueIndex;size:255"` // Format: "GOOGLE:123456789"

    // Captain Specific Fields
    VehicleType       string  `gorm:"size:50"` // BIKE / CAR / VAN / TRUCK
    VehicleNumber     string  `gorm:"size:50"`
    VehicleModel      string  `gorm:"size:100"`
    LicenseNumber     string  `gorm:"size:100"`
    LicenseExpiry     *time.Time
    IsAvailable       bool    `gorm:"default:false;index"` // Is captain currently available for orders
    CurrentLat        float64 `gorm:"type:decimal(10,8)"` // Current location latitude
    CurrentLng        float64 `gorm:"type:decimal(11,8)"` // Current location longitude
    CurrentLocationUpdatedAt *time.Time

    // Ratings & Stats
    Rating           float64 `gorm:"type:decimal(3,2);default:0"` // Average rating (0-5)
    TotalRatings     int     `gorm:"default:0"`
    TotalDeliveries  int     `gorm:"default:0"` // For captains
    TotalOrders      int     `gorm:"default:0"` // For customers
    CompletionRate   float64 `gorm:"type:decimal(5,2);default:0"` // For captains
    CancellationRate float64 `gorm:"type:decimal(5,2);default:0"` // For both

    // KYC & Verification (for Captains)
    KycStatus         string `gorm:"default:'PENDING'"` // PENDING / VERIFIED / REJECTED
    KycDocuments      string `gorm:"type:text"` // JSON array of document URLs
    BackgroundCheckStatus string `gorm:"default:'PENDING'"` // PENDING / VERIFIED / FAILED
    OnboardingCompletedAt *time.Time

    // Financial
    WalletBalance float64 `gorm:"type:decimal(10,2);default:0"`
    Currency      string  `gorm:"default:'INR';size:3"`

    // Preferences (stored as JSON)
    Preferences *string `gorm:"type:jsonb"` // Notification settings, language, etc.

    // Metadata
    LastActiveAt    *time.Time
    DeviceToken     string `gorm:"size:500"` // For push notifications
    DevicePlatform  string `gorm:"size:20"`  // IOS / ANDROID / WEB
    AppVersion      string `gorm:"size:20"`
    ReferralCode    *string `gorm:"uniqueIndex;size:20"`
    ReferredBy      *uint  // User ID who referred this user

    // Relationships
    Addresses        []Address       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
    Orders           []Order         `gorm:"foreignKey:CustomerID;constraint:OnDelete:SET NULL"`
    DeliveredOrders  []Order         `gorm:"foreignKey:CaptainID;constraint:OnDelete:SET NULL"`
    Ratings          []Rating        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
    Transactions     []Transaction   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
    RefreshTokens    []RefreshToken  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name
func (User) TableName() string {
    return "users"
}

// GetFullName returns user's full name
func (u *User) GetFullName() string {
    if u.FirstName == "" && u.LastName == "" {
        return "User"
    }
    return u.FirstName + " " + u.LastName
}

// IsCaptain checks if user is a captain
func (u *User) IsCaptain() bool {
    return u.Role == "CAPTAIN"
}

// IsCustomer checks if user is a customer
func (u *User) IsCustomer() bool {
    return u.Role == "CUSTOMER"
}

// IsActive checks if user account is active
func (u *User) IsActive() bool {
    return u.Status == "ACTIVE"
}

// CanTakeOrders checks if captain can accept new orders
func (u *User) CanTakeOrders() bool {
    return u.IsCaptain() && u.IsActive() && u.IsAvailable && u.KycStatus == "VERIFIED"
}