package postgres

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// ============================================================================
// DATABASE MODELS (GORM)
// These are separate from domain models to maintain clean architecture
// ============================================================================

// UserModel represents the users table
type UserModel struct {
	ID      string `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Version int    `gorm:"not null;default:1"`

	// Profile
	FirstName  string `gorm:"size:100"`
	LastName   string `gorm:"size:100"`
	ProfilePic string `gorm:"size:500"`

	// Contact
	Email         *string `gorm:"size:255;uniqueIndex:idx_users_email"`
	EmailVerified bool    `gorm:"not null;default:false"`
	Phone         *string `gorm:"size:20;uniqueIndex:idx_users_phone"`
	PhoneVerified bool    `gorm:"not null;default:false"`

	// Role & Status
	PrimaryRole string         `gorm:"size:20;not null;index:idx_users_primary_role"`
	Roles       pq.StringArray `gorm:"type:text[];not null;default:'{}'"`
	Status      string         `gorm:"size:20;not null;default:'ACTIVE';index:idx_users_status"`

	// Authentication
	Provider   string  `gorm:"size:50;index:idx_users_provider"`
	ProviderID *string `gorm:"size:255;uniqueIndex"`

	// Wallet
	WalletBalance  int64  `gorm:"not null;default:0"`
	WalletCurrency string `gorm:"size:3;not null;default:'INR'"`
	WalletVersion  int    `gorm:"not null;default:1"`

	// Stats
	TotalOrders     int     `gorm:"not null;default:0"`
	TotalDeliveries int     `gorm:"not null;default:0"`
	Rating          float64 `gorm:"type:decimal(3,2);default:0.00"`
	TotalRatings    int     `gorm:"not null;default:0"`

	// Metadata
	LastActiveAt *time.Time
	CreatedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt    *time.Time `gorm:"index"`

	// Relationships
	CaptainProfile *CaptainProfileModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	AdminProfile   *AdminProfileModel   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Addresses      []AddressModel       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (UserModel) TableName() string {
	return "users"
}

// CaptainProfileModel represents the captain_profiles table
type CaptainProfileModel struct {
	ID      uint   `gorm:"primaryKey"`
	UserID  string `gorm:"type:uuid;not null;uniqueIndex"`
	Version int    `gorm:"not null;default:1"`

	// Vehicle
	VehicleType   *string `gorm:"size:20"`
	VehicleNumber *string `gorm:"size:50;uniqueIndex"`
	VehicleModel  *string `gorm:"size:100"`

	// License
	LicenseNumber *string `gorm:"size:100;uniqueIndex"`
	LicenseExpiry *time.Time

	// KYC
	KYCStatus    string `gorm:"size:20;not null;default:'PENDING';index:idx_captain_kyc_status"`
	KYCDocuments JSONB  `gorm:"type:jsonb;not null;default:'{}'"`

	// Operational
	IsAvailable       bool    `gorm:"not null;default:false;index:idx_captain_availability"`
	CurrentLocation   *string `gorm:"type:geography(POINT,4326)"` // PostGIS geography
	LocationUpdatedAt *time.Time

	// Performance
	CompletionRate   float64 `gorm:"type:decimal(5,2);default:0.00"`
	CancellationRate float64 `gorm:"type:decimal(5,2);default:0.00"`
	OnTimeRate       float64 `gorm:"type:decimal(5,2);default:0.00"`

	// Timestamps
	OnboardedAt *time.Time
	VerifiedAt  *time.Time
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt   *time.Time
}

func (CaptainProfileModel) TableName() string {
	return "captain_profiles"
}

// AdminProfileModel represents the admin_profiles table
type AdminProfileModel struct {
	ID      uint   `gorm:"primaryKey"`
	UserID  string `gorm:"type:uuid;not null;uniqueIndex"`
	Version int    `gorm:"not null;default:1"`

	// Permissions
	Permissions pq.StringArray `gorm:"type:text[];not null;default:'{}'"`

	// Department
	Department string  `gorm:"size:50;not null;index:idx_admin_department"`
	EmployeeID *string `gorm:"size:50;uniqueIndex"`

	// Timestamps
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt *time.Time
}

func (AdminProfileModel) TableName() string {
	return "admin_profiles"
}

// AddressModel represents the addresses table
type AddressModel struct {
	ID      string `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID  string `gorm:"type:uuid;not null;index:idx_addresses_user_id"`
	Version int    `gorm:"not null;default:1"`

	// Basic Information
	Label        *string `gorm:"size:20"`
	AddressLine1 string  `gorm:"size:255;not null"`
	AddressLine2 *string `gorm:"size:255"`
	Landmark     *string `gorm:"size:255"`
	City         string  `gorm:"size:100;not null;index:idx_addresses_city"`
	State        string  `gorm:"size:100;not null;index:idx_addresses_state"`
	Country      string  `gorm:"size:100;not null;default:'India'"`
	PostalCode   *string `gorm:"size:20;index:idx_addresses_postal_code"`

	// Geolocation (PostGIS)
	Location string `gorm:"type:geography(POINT,4326);not null"`

	// Contact
	ContactName  *string `gorm:"size:100"`
	ContactPhone *string `gorm:"size:20"`

	// Status
	IsDefault  bool `gorm:"not null;default:false"`
	IsActive   bool `gorm:"not null;default:true;index:idx_addresses_is_active"`
	IsVerified bool `gorm:"not null;default:false"`

	// Usage Analytics
	UsageCount int        `gorm:"not null;default:0"`
	LastUsedAt *time.Time `gorm:"index:idx_addresses_last_used"`

	// Time-based Analytics
	MorningUsageCount   int `gorm:"not null;default:0"`
	AfternoonUsageCount int `gorm:"not null;default:0"`
	EveningUsageCount   int `gorm:"not null;default:0"`
	NightUsageCount     int `gorm:"not null;default:0"`

	// Day-based Analytics
	WeekdayUsageCount int `gorm:"not null;default:0"`
	WeekendUsageCount int `gorm:"not null;default:0"`

	// Monthly Analytics (JSONB)
	MonthlyUsageCount JSONB `gorm:"type:jsonb;not null;default:'{}'"`

	// Timestamps
	CreatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt *time.Time `gorm:"index"`
}

func (AddressModel) TableName() string {
	return "addresses"
}

// OTPSessionModel represents the otp_sessions table
type OTPSessionModel struct {
	ID        uint      `gorm:"primaryKey"`
	Target    string    `gorm:"size:255;not null;index:idx_otp_target_purpose"`
	Code      string    `gorm:"size:10;not null"`
	Purpose   string    `gorm:"size:50;not null;index:idx_otp_target_purpose"`
	Used      bool      `gorm:"not null;default:false"`
	Attempts  int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	ExpiresAt time.Time `gorm:"not null;index:idx_otp_expires_at"`
}

func (OTPSessionModel) TableName() string {
	return "otp_sessions"
}

// RefreshTokenModel represents the refresh_tokens table
type RefreshTokenModel struct {
	ID         uint    `gorm:"primaryKey"`
	UserID     string  `gorm:"type:uuid;not null;index:idx_refresh_tokens_user_id"`
	Token      string  `gorm:"size:500;not null;uniqueIndex"`
	Device     *string `gorm:"size:50"`
	DeviceInfo JSONB   `gorm:"type:jsonb"`

	IsRevoked bool `gorm:"not null;default:false"`
	RevokedAt *time.Time

	CreatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	ExpiresAt  time.Time `gorm:"not null;index:idx_refresh_tokens_expires_at"`
	LastUsedAt *time.Time
}

func (RefreshTokenModel) TableName() string {
	return "refresh_tokens"
}

// ============================================================================
// JSONB TYPE (for PostgreSQL JSONB columns)
// ============================================================================

type JSONB map[string]interface{}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return json.Marshal(j)
}
