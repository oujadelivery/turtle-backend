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
	FindByIDs(ctx context.Context, ids []string) ([]*aggregates.User, error)

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
	FindByIDs(ctx context.Context, ids []string) ([]*aggregates.Address, error)

	// FindByUserID finds all addresses for a user
	FindByUserID(ctx context.Context, userID string) ([]*aggregates.Address, error)
	FindByUserIDs(ctx context.Context, userIDs []string) (map[string][]*aggregates.Address, error)

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



// ============================================================================
// ORDER REPOSITORY
// ============================================================================

// OrderRepository manages order persistence
type OrderRepository interface {
	// Single operations
	Create(ctx context.Context, order *aggregates.Order) error
	Update(ctx context.Context, order *aggregates.Order) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*aggregates.Order, error)

	// Batch operations (for DataLoader)
	FindByIDs(ctx context.Context, ids []string) ([]*aggregates.Order, error)

	// User-related queries
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*aggregates.Order, int64, error)
	FindActiveCaptainOrders(ctx context.Context, captainID string) ([]*aggregates.Order, error)
	FindByCaptainID(ctx context.Context, captainID string, limit, offset int) ([]*aggregates.Order, int64, error)

	// State-based queries
	FindByState(ctx context.Context, state aggregates.OrderState, limit, offset int) ([]*aggregates.Order, int64, error)
	FindActiveOrders(ctx context.Context, limit, offset int) ([]*aggregates.Order, int64, error)
	FindOrdersInStates(ctx context.Context, states []aggregates.OrderState, limit, offset int) ([]*aggregates.Order, int64, error)

	// Geospatial queries for captain matching
	FindNearbyOrders(ctx context.Context, latitude, longitude, radiusKm float64, state aggregates.OrderState) ([]*aggregates.Order, error)
	FindSearchingCaptainNearby(ctx context.Context, latitude, longitude, radiusKm float64) ([]*aggregates.Order, error)

	// Scheduled orders
	FindScheduledOrders(ctx context.Context, before time.Time, limit int) ([]*aggregates.Order, error)

	// Payment-related queries
	FindPendingPaymentCapture(ctx context.Context, limit int) ([]*aggregates.Order, error)

	// Statistics
	CountCompletedByUser(ctx context.Context, userID string) (int64, error)
	CountCompletedByCaptain(ctx context.Context, captainID string) (int64, error)
	CountByUserInTimeRange(ctx context.Context, userID string, startTime, endTime time.Time) (int64, error)

	// Admin queries
	FindAll(ctx context.Context, limit, offset int) ([]*aggregates.Order, int64, error)
	SearchOrders(ctx context.Context, query string, limit, offset int) ([]*aggregates.Order, int64, error)
}

// ============================================================================
// ORDER STATE HISTORY REPOSITORY
// ============================================================================

// OrderStateHistoryRepository manages order state transition history
type OrderStateHistoryRepository interface {
	Create(ctx context.Context, history *OrderStateHistory) error
	FindByOrderID(ctx context.Context, orderID string) ([]*OrderStateHistory, error)
	FindByTriggeredBy(ctx context.Context, userID string, limit, offset int) ([]*OrderStateHistory, int64, error)
}

// OrderStateHistory represents a state transition record
type OrderStateHistory struct {
	ID          int64
	OrderID     string
	FromState   string
	ToState     string
	TriggeredBy string
	Reason      string
	CreatedAt   time.Time
}

// ============================================================================
// ORDER EVENT REPOSITORY (for Event Sourcing - Future)
// ============================================================================

// OrderEventRepository manages order events for event sourcing
type OrderEventRepository interface {
	Create(ctx context.Context, event *OrderEvent) error
	FindByOrderID(ctx context.Context, orderID string) ([]*OrderEvent, error)
	FindByAggregateVersion(ctx context.Context, orderID string, version int) (*OrderEvent, error)
}

// OrderEvent represents a domain event for event sourcing
type OrderEvent struct {
	ID               int64
	OrderID          string
	EventType        string
	EventData        []byte // JSON-encoded event data
	AggregateVersion int
	CreatedAt        time.Time
}

// ============================================================================
// CAPTAIN ASSIGNMENT REPOSITORY
// ============================================================================

// CaptainAssignmentRepository manages captain assignment attempts
type CaptainAssignmentRepository interface {
	CreateAttempt(ctx context.Context, orderID, captainID string, attemptNumber int, timeoutAt time.Time) error
	MarkAccepted(ctx context.Context, orderID, captainID string) error
	MarkDeclined(ctx context.Context, orderID, captainID string, reason string) error
	MarkTimeout(ctx context.Context, orderID, captainID string) error
	
	FindByOrderID(ctx context.Context, orderID string) ([]*CaptainAssignmentAttempt, error)
	FindByCaptainID(ctx context.Context, captainID string, limit, offset int) ([]*CaptainAssignmentAttempt, int64, error)
	FindTimedOutAttempts(ctx context.Context, before time.Time, limit int) ([]*CaptainAssignmentAttempt, error)
	
	CountAttemptsByOrder(ctx context.Context, orderID string) (int, error)
}

// CaptainAssignmentAttempt represents a captain assignment attempt
type CaptainAssignmentAttempt struct {
	ID            int64
	OrderID       string
	CaptainID     string
	AttemptNumber int
	Status        string // SENT, ACCEPTED, DECLINED, TIMEOUT
	DeclineReason string
	SentAt        time.Time
	RespondedAt   *time.Time
	TimeoutAt     time.Time
}

// ============================================================================
// CANCELLATION EVENT REPOSITORY
// ============================================================================

// CancellationEventRepository manages cancellation event records
type CancellationEventRepository interface {
	Create(ctx context.Context, event *CancellationEvent) error
	FindByID(ctx context.Context, id int64) (*CancellationEvent, error)
	FindByOrderID(ctx context.Context, orderID string) ([]*CancellationEvent, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*CancellationEvent, int64, error)
	FindByRole(ctx context.Context, role string, limit, offset int) ([]*CancellationEvent, int64, error)
	FindInTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]*CancellationEvent, int64, error)
	
	// Analytics
	CountByUser(ctx context.Context, userID string) (int64, error)
	CountByUserInTimeRange(ctx context.Context, userID string, startTime, endTime time.Time) (int64, error)
}

// ============================================================================
// USER PENALTY REPOSITORY
// ============================================================================

// UserPenaltyRepository manages user penalty and reward points
type UserPenaltyRepository interface {
	Create(ctx context.Context, penalty *UserPenalty) error
	Update(ctx context.Context, penalty *UserPenalty) error
	FindByID(ctx context.Context, id int64) (*UserPenalty, error)
	FindByUserID(ctx context.Context, userID string) (*UserPenalty, error)
	
	// Batch operations (for DataLoader)
	FindByUserIDs(ctx context.Context, userIDs []string) ([]*UserPenalty, error)
	
	// Suspended users
	FindSuspendedUsers(ctx context.Context, limit, offset int) ([]*UserPenalty, int64, error)
	FindExpiredSuspensions(ctx context.Context, before time.Time, limit int) ([]*UserPenalty, error)
	
	// Leaderboard
	FindTopRewardUsers(ctx context.Context, limit int) ([]*UserPenalty, error)
	FindTopPenaltyUsers(ctx context.Context, limit int) ([]*UserPenalty, error)
}

// ============================================================================
// TRANSACTION REPOSITORY
// ============================================================================

// TransactionRepository manages payment transactions
type TransactionRepository interface {
	Create(ctx context.Context, tx *Transaction) error
	Update(ctx context.Context, tx *Transaction) error
	FindByID(ctx context.Context, id int64) (*Transaction, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*Transaction, int64, error)
	FindByOrderID(ctx context.Context, orderID string) ([]*Transaction, error)
	
	// Payment intent queries
	FindByPaymentIntentID(ctx context.Context, intentID string) (*Transaction, error)
	FindPendingTransactions(ctx context.Context, limit, offset int) ([]*Transaction, int64, error)
	
	// Analytics
	SumByUser(ctx context.Context, userID string, txType string) (int64, error)
	CountByUser(ctx context.Context, userID string) (int64, error)
}


// ============================================================================
// PRICING RULE REPOSITORY (Future)
// ============================================================================

// PricingRuleRepository manages dynamic pricing rules
type PricingRuleRepository interface {
	FindActiveRules(ctx context.Context) ([]*PricingRule, error)
	FindByCity(ctx context.Context, city string) ([]*PricingRule, error)
	FindSurgeZones(ctx context.Context, latitude, longitude float64) ([]*SurgeZone, error)
}

// PricingRule represents a pricing configuration
type PricingRule struct {
	ID               int64
	Name             string
	BasePrice        int64 // in paise
	PricePerKm       int64 // in paise
	WeightThreshold  float64
	PricePerKgAbove  int64
	City             string
	IsActive         bool
	EffectiveFrom    time.Time
	EffectiveUntil   *time.Time
}

// SurgeZone represents a surge pricing zone
type SurgeZone struct {
	ID             int64
	Name           string
	Latitude       float64
	Longitude      float64
	RadiusKm       float64
	SurgeMultiplier float64
	IsActive       bool
	ActiveFrom     time.Time
	ActiveUntil    *time.Time
}

// ============================================================================
// PROMO CODE REPOSITORY (Future)
// ============================================================================

// PromoCodeRepository manages promotional discount codes
type PromoCodeRepository interface {
	FindByCode(ctx context.Context, code string) (*PromoCode, error)
	FindActivePromoCodes(ctx context.Context) ([]*PromoCode, error)
	ValidatePromoCode(ctx context.Context, code string, userID string, orderAmount int64) (bool, error)
	IncrementUsage(ctx context.Context, code string, userID string) error
}

// PromoCode represents a promotional discount code
type PromoCode struct {
	ID                int64
	Code              string
	DiscountType      string // PERCENTAGE, FIXED_AMOUNT
	DiscountValue     int64  // percentage (0-100) or amount in paise
	MinOrderAmount    int64  // in paise
	MaxDiscountAmount int64  // in paise
	MaxUsagePerUser   int
	TotalUsageLimit   int
	CurrentUsageCount int
	IsActive          bool
	ValidFrom         time.Time
	ValidUntil        time.Time
}

// ============================================================================
// NOTIFICATION REPOSITORY (Future)
// ============================================================================

// NotificationRepository manages notification delivery tracking
type NotificationRepository interface {
	Create(ctx context.Context, notification *Notification) error
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*Notification, int64, error)
	FindUnread(ctx context.Context, userID string) ([]*Notification, error)
	MarkAsRead(ctx context.Context, id int64) error
	MarkAllAsRead(ctx context.Context, userID string) error
}

// Notification represents a notification record
type Notification struct {
	ID        int64
	UserID    string
	Type      string // ORDER_UPDATE, CAPTAIN_ASSIGNED, DELIVERY_COMPLETED, etc.
	Title     string
	Message   string
	OrderID   *string
	IsRead    bool
	ReadAt    *time.Time
	CreatedAt time.Time
}

// ============================================================================
// RATING REPOSITORY (Future)
// ============================================================================

// RatingRepository manages order ratings and reviews
type RatingRepository interface {
	Create(ctx context.Context, rating *Rating) error
	Update(ctx context.Context, rating *Rating) error
	FindByID(ctx context.Context, id int64) (*Rating, error)
	FindByOrderID(ctx context.Context, orderID string) (*Rating, error)
	FindByCaptainID(ctx context.Context, captainID string, limit, offset int) ([]*Rating, int64, error)
	FindByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*Rating, int64, error)
	
	// Analytics
	CalculateAverageRating(ctx context.Context, userID string, asRole string) (float64, error)
	CountRatings(ctx context.Context, userID string) (int64, error)
}

// Rating represents an order rating
type Rating struct {
	ID               int64
	OrderID          string
	RatedByUserID    string // Who gave the rating
	RatedUserID      string // Who received the rating
	RatedUserRole    string // CUSTOMER or CAPTAIN
	Score            float64 // 1.0 to 5.0
	Review           string
	Tags             []string // helpful, polite, professional, etc.
	CreatedAt        time.Time
	UpdatedAt        time.Time
}