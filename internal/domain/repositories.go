package domain

import (
	"context"
	"time"
	"turtle/internal/domain/aggregates"
)

// UserRepository defines the interface for user persistence
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *aggregates.User) error

	// FindByID finds a user by ID
	FindByID(ctx context.Context, id string) (*aggregates.User, error)

	// FindByEmail finds a user by email
	FindByEmail(ctx context.Context, email string) (*aggregates.User, error)

	// FindByPhone finds a user by phone
	FindByPhone(ctx context.Context, phone string) (*aggregates.User, error)

	// FindByProviderID finds a user by provider ID (for social login)
	FindByProviderID(ctx context.Context, provider, providerID string) (*aggregates.User, error)

	// Update updates a user (with optimistic locking)
	Update(ctx context.Context, user *aggregates.User) error

	// UpdateWallet updates user wallet balance (with optimistic locking)
	UpdateWallet(ctx context.Context, userID string, expectedVersion int64, newBalance int64) error

	// Delete soft deletes a user
	Delete(ctx context.Context, id string) error

	// FindCaptainsNearby finds available captains within radius
	FindCaptainsNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*aggregates.User, error)

	// FindCaptainsByStatus finds captains by KYC status
	FindCaptainsByStatus(ctx context.Context, status string) ([]*aggregates.User, error)

	// UpdateCaptainLocation updates captain's current location
	UpdateCaptainLocation(ctx context.Context, captainID string, lat, lng float64) error

	// Search searches users by name, email, or phone
	Search(ctx context.Context, query string, role string, limit, offset int) ([]*aggregates.User, int64, error)

	// HealthCheck verifies database connectivity
	HealthCheck(ctx context.Context) error

	// GetUserStats returns aggregated user statistics
	GetUserStats(ctx context.Context) (*UserStats, error)
}

// UserStats represents aggregated user statistics
type UserStats struct {
	TotalUsers       int64
	TotalCustomers   int64
	TotalCaptains    int64
	TotalAdmins      int64
	ActiveUsers      int64
	VerifiedCaptains int64
}

// AddressRepository defines the interface for address persistence
type AddressRepository interface {
	// Create creates a new address
	Create(ctx context.Context, address *aggregates.Address) error

	// FindByID finds an address by ID
	FindByID(ctx context.Context, id string) (*aggregates.Address, error)

	// FindByUserID finds all addresses for a user
	FindByUserID(ctx context.Context, userID string) ([]*aggregates.Address, error)

	// FindDefaultByUserID finds user's default address
	FindDefaultByUserID(ctx context.Context, userID string) (*aggregates.Address, error)

	// Update updates an address
	Update(ctx context.Context, address *aggregates.Address) error

	// Delete soft deletes an address
	Delete(ctx context.Context, id string) error

	// UnsetDefault removes default flag from all user's addresses
	UnsetDefault(ctx context.Context, userID string) error

	// IncrementUsage increments address usage statistics
	IncrementUsage(ctx context.Context, addressID string) error

	// FindSuggestedAddresses finds addresses that should be suggested at current time
	FindSuggestedAddresses(ctx context.Context, userID string, limit int) ([]*aggregates.Address, error)

	// FindNearest finds nearest addresses to given location
	FindNearest(ctx context.Context, userID string, lat, lng float64, limit int) ([]*aggregates.Address, error)

	// SearchAddresses searches addresses with pagination
	SearchAddresses(ctx context.Context, userID string, query string, limit, offset int) ([]*aggregates.Address, int64, error)

	// GetAddressStats returns address statistics for a user
	GetAddressStats(ctx context.Context, userID string) (*AddressStats, error)
}

// AddressStats represents address usage statistics
type AddressStats struct {
	TotalAddresses  int64
	MostUsedAddress *aggregates.Address
	DefaultAddress  *aggregates.Address
	RecentAddresses []*aggregates.Address
}

// OTPRepository defines the interface for OTP session persistence
type OTPRepository interface {
	// Create creates a new OTP session
	Create(ctx context.Context, target, code, purpose string, expiresAt time.Time) error

	// FindByTarget finds the latest OTP session for target and purpose
	FindByTarget(ctx context.Context, target, purpose string) (*OTPSession, error)

	// MarkAsUsed marks an OTP session as used
	MarkAsUsed(ctx context.Context, target, purpose string) error

	// IncrementAttempts increments failed verification attempts
	IncrementAttempts(ctx context.Context, target, purpose string) error

	// DeleteExpired deletes expired OTP sessions
	DeleteExpired(ctx context.Context) error
}

// OTPSession represents an OTP session (for repository)
type OTPSession struct {
	ID        uint
	Target    string
	Code      string
	Purpose   string
	Used      bool
	Attempts  int
	CreatedAt time.Time
	ExpiresAt time.Time
}

// RefreshTokenRepository defines the interface for refresh token persistence
type RefreshTokenRepository interface {
	// Create creates a new refresh token
	Create(ctx context.Context, userID, token, device string, expiresAt time.Time) error

	// FindByToken finds a refresh token
	FindByToken(ctx context.Context, token string) (*RefreshToken, error)

	// FindByUserID finds all refresh tokens for a user
	FindByUserID(ctx context.Context, userID string) ([]*RefreshToken, error)

	// FindActiveByUserID finds all active (non-revoked) refresh tokens for a user
	FindActiveByUserID(ctx context.Context, userID string) ([]*RefreshToken, error)

	// UpdateLastUsed updates the last used timestamp
	UpdateLastUsed(ctx context.Context, token string) error

	// Revoke revokes a refresh token
	Revoke(ctx context.Context, token string) error

	// RevokeAllForUser revokes all refresh tokens for a user
	RevokeAllForUser(ctx context.Context, userID string) error

	// DeleteExpired deletes expired refresh tokens
	DeleteExpired(ctx context.Context) error
}

// RefreshToken represents a refresh token (for repository)
type RefreshToken struct {
	ID         uint
	UserID     string
	Token      string
	Device     string
	DeviceInfo map[string]interface{}
	IsRevoked  bool
	RevokedAt  *time.Time
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastUsedAt *time.Time
}
// GetUsagePatterns returns usage patterns for ML/analytics
type UsagePattern struct {
	AddressID           string
	TotalUsage          int
	MorningPercentage   float64
	AfternoonPercentage float64
	EveningPercentage   float64
	NightPercentage     float64
	WeekdayPercentage   float64
	WeekendPercentage   float64
	AverageGap          float64 // Average days between uses
}
