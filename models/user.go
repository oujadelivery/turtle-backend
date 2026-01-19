package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// JSONB is a custom type for PostgreSQL JSONB fields
type JSONB map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}

	return json.Unmarshal(bytes, j)
}

// User model with fixed JSON fields
type User struct {
	gorm.Model

	// Basic Info
	FirstName  string `gorm:"size:100;index:idx_name"`
	LastName   string `gorm:"size:100;index:idx_name"`
	ProfilePic string `gorm:"size:500"`

	// Contact
	Email         *string `gorm:"uniqueIndex;size:255"`
	EmailVerified bool    `gorm:"default:false"`
	Phone         *string `gorm:"uniqueIndex;size:20"`
	PhoneVerified bool    `gorm:"default:false"`

	// Role & Status
	Role   string `gorm:"not null;default:'CUSTOMER';index:idx_role_status"`
	Status string `gorm:"default:'ACTIVE';index:idx_role_status"`

	// Social Login
	Provider   string  `gorm:"size:50;index"`
	ProviderID *string `gorm:"uniqueIndex;size:255"` // Format: "GOOGLE:123456789"

	// Captain Specific Fields
	VehicleType   *string    `gorm:"size:50;index"`
	VehicleNumber *string    `gorm:"size:50;uniqueIndex"`
	VehicleModel  *string    `gorm:"size:100"`
	LicenseNumber *string    `gorm:"size:100;uniqueIndex"`
	LicenseExpiry *time.Time `gorm:"index"`

	IsAvailable              bool       `gorm:"default:false;index:idx_available_captain"`
	CurrentLat               float64    `gorm:"type:decimal(10,8);index:idx_captain_location"`
	CurrentLng               float64    `gorm:"type:decimal(11,8);index:idx_captain_location"`
	CurrentLocationUpdatedAt *time.Time `gorm:"index"`

	// Ratings & Stats
	Rating           float64 `gorm:"type:decimal(3,2);default:0;index"`
	TotalRatings     int     `gorm:"default:0"`
	TotalDeliveries  int     `gorm:"default:0;index"`
	TotalOrders      int     `gorm:"default:0;index"`
	CompletionRate   float64 `gorm:"type:decimal(5,2);default:0;index"`
	CancellationRate float64 `gorm:"type:decimal(5,2);default:0"`

	// KYC & Verification
	KycStatus             string     `gorm:"default:'PENDING';index"`
	KycDocuments          JSONB      `gorm:"type:jsonb"` // FIXED: Use JSONB type
	BackgroundCheckStatus string     `gorm:"default:'PENDING';index"`
	OnboardingCompletedAt *time.Time `gorm:"index"`

	// Financial
	WalletBalance float64 `gorm:"type:decimal(10,2);default:0"`
	Currency      string  `gorm:"default:'INR';size:3"`
	WalletVersion int     `gorm:"default:1"`

	// Preferences - FIXED: Use JSONB type
	Preferences JSONB `gorm:"type:jsonb"`

	// Metadata
	LastActiveAt   *time.Time `gorm:"index:idx_last_active"`
	DeviceToken    string     `gorm:"size:500"`
	DevicePlatform string     `gorm:"size:20"`
	AppVersion     string     `gorm:"size:20"`
	ReferralCode   *string    `gorm:"uniqueIndex;size:20"`
	ReferredBy     *uint

	// Security fields
	LastLoginAt         *time.Time
	LastLoginIP         string `gorm:"size:45"`
	LoginCount          int    `gorm:"default:0"`
	FailedLoginAttempts int    `gorm:"default:0"`
	LockedUntil         *time.Time
	IsSuspicious        bool `gorm:"default:false;index"`

	// Relationships
	Addresses       []Address      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Orders          []Order        `gorm:"foreignKey:CustomerID;constraint:OnDelete:SET NULL"`
	DeliveredOrders []Order        `gorm:"foreignKey:CaptainID;constraint:OnDelete:SET NULL"`
	Ratings         []Rating       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Transactions    []Transaction  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	RefreshTokens   []RefreshToken `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook to initialize JSON fields
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Initialize JSONB fields if nil
	if u.KycDocuments == nil {
		u.KycDocuments = JSONB{}
	}
	if u.Preferences == nil {
		u.Preferences = JSONB{}
	}
	return nil
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

// IsLocked checks if account is locked
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// SetPreference sets a user preference
func (u *User) SetPreference(key string, value interface{}) {
	if u.Preferences == nil {
		u.Preferences = JSONB{}
	}
	u.Preferences[key] = value
}

// GetPreference gets a user preference
func (u *User) GetPreference(key string) (interface{}, bool) {
	if u.Preferences == nil {
		return nil, false
	}
	val, ok := u.Preferences[key]
	return val, ok
}

// AddKYCDocument adds a KYC document
func (u *User) AddKYCDocument(docType, url string) {
	if u.KycDocuments == nil {
		u.KycDocuments = JSONB{}
	}
	u.KycDocuments[docType] = url
}

// GetKYCDocument gets a KYC document URL
func (u *User) GetKYCDocument(docType string) (string, bool) {
	if u.KycDocuments == nil {
		return "", false
	}
	if url, ok := u.KycDocuments[docType].(string); ok {
		return url, true
	}
	return "", false
}
