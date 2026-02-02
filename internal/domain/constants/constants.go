package constants

import "time"

// ============================================================================
// AUTHENTICATION CONSTANTS
// ============================================================================

// OTP Configuration
const (
	// OTPLength is the length of OTP codes for login/verification
	OTPLength = 6

	// OTPExpiryDuration is how long an OTP remains valid
	OTPExpiryDuration = 15 * time.Minute

	// OTPMaxAttempts is the maximum number of failed OTP verification attempts
	OTPMaxAttempts = 5

	// OrderOTPLength is the length of order-related OTPs (alphanumeric)
	OrderOTPLength = 6

	// OrderOTPExpiry is how long order OTPs remain valid
	OrderOTPExpiry = 24 * time.Hour
)

// Rate Limiting
const (
	// OTPSendLimit is the maximum number of OTP requests per phone/email
	OTPSendLimit = 3

	// OTPSendWindow is the time window for OTP send rate limiting
	OTPSendWindow = 1 * time.Hour

	// OTPVerifyLimit is the maximum verification attempts in time window
	OTPVerifyLimit = 5

	// OTPVerifyWindow is the time window for verification rate limiting
	OTPVerifyWindow = 15 * time.Minute

	// LoginAttemptsLimit is the maximum login attempts per email/IP
	LoginAttemptsLimit = 10

	// LoginAttemptsWindow is the time window for login rate limiting
	LoginAttemptsWindow = 1 * time.Hour
)

// JWT Configuration
const (
	// AccessTokenDuration is the lifetime of an access token
	AccessTokenDuration = 15 * time.Minute

	// RefreshTokenDuration is the lifetime of a refresh token
	RefreshTokenDuration = 60 * 24 * time.Hour // 60 days
)

// ============================================================================
// DATABASE CONSTANTS
// ============================================================================

// Connection Pool
const (
	// DBMaxOpenConns is the maximum number of open database connections
	DBMaxOpenConns = 25

	// DBMaxIdleConns is the maximum number of idle database connections
	DBMaxIdleConns = 5

	// DBConnMaxLifetime is the maximum lifetime of a database connection
	DBConnMaxLifetime = 5 * time.Minute

	// DBConnMaxIdleTime is the maximum idle time for a database connection
	DBConnMaxIdleTime = 10 * time.Minute
)

// Query Timeouts
const (
	// DBQueryTimeout is the default timeout for database queries
	DBQueryTimeout = 10 * time.Second

	// DBTransactionTimeout is the timeout for database transactions
	DBTransactionTimeout = 30 * time.Second
)

// ============================================================================
// CACHE CONSTANTS
// ============================================================================

// Redis Configuration
const (
	// CacheDefaultExpiry is the default cache expiration time
	CacheDefaultExpiry = 1 * time.Hour

	// CacheShortExpiry is for frequently changing data
	CacheShortExpiry = 5 * time.Minute

	// CacheLongExpiry is for rarely changing data
	CacheLongExpiry = 24 * time.Hour
)

// Lock Configuration
const (
	// LockDefaultTTL is the default TTL for distributed locks
	LockDefaultTTL = 30 * time.Second

	// LockMaxRetries is the maximum number of lock acquisition retries
	LockMaxRetries = 3

	// LockRetryDelay is the base delay between lock acquisition retries
	LockRetryDelay = 100 * time.Millisecond
)

// ============================================================================
// BUSINESS LOGIC CONSTANTS
// ============================================================================

// Captain Configuration
const (
	// CaptainLocationUpdateInterval is how often captain location should be updated
	CaptainLocationUpdateInterval = 30 * time.Second

	// CaptainOfflineThreshold is the time after which captain is considered offline
	CaptainOfflineThreshold = 2 * time.Minute

	// CaptainSearchRadius is the default search radius for nearby captains (in km)
	CaptainSearchRadius = 5.0

	// CaptainMaxActiveOrders is the maximum concurrent orders per captain
	CaptainMaxActiveOrders = 1
)

// Order Configuration
const (
	// OrderDefaultWaitingTime is the default waiting time in minutes
	OrderDefaultWaitingTime = 5

	// OrderWaitingChargePerMinute is the charge per minute of waiting (in paise)
	OrderWaitingChargePerMinute = 200 // ₹2.00

	// OrderCancellationWindow is the time window for free cancellation
	OrderCancellationWindow = 5 * time.Minute

	// OrderPickupOTPExpiry is the expiry time for pickup OTP
	OrderPickupOTPExpiry = 24 * time.Hour

	// OrderDeliveryOTPExpiry is the expiry time for delivery OTP
	OrderDeliveryOTPExpiry = 24 * time.Hour
)

// Pricing Configuration
const (
	// BasePriceINR is the base price for delivery in paise
	BasePriceINR = 5000 // ₹50.00

	// PricePerKmINR is the price per kilometer in paise
	PricePerKmINR = 1000 // ₹10.00

	// PlatformCommissionPercent is the platform commission percentage
	PlatformCommissionPercent = 20.0

	// InsuranceThresholdINR is the minimum parcel value for insurance in paise
	InsuranceThresholdINR = 500000 // ₹5000.00

	// InsuranceFeePercent is the insurance fee percentage
	InsuranceFeePercent = 1.0
)

// Wallet Configuration
const (
	// MinWalletBalanceINR is the minimum wallet balance in paise
	MinWalletBalanceINR = 0

	// MaxWalletBalanceINR is the maximum wallet balance in paise
	MaxWalletBalanceINR = 10000000 // ₹100,000

	// WalletTopupMinINR is the minimum wallet top-up amount in paise
	WalletTopupMinINR = 10000 // ₹100

	// WalletTopupMaxINR is the maximum wallet top-up amount in paise
	WalletTopupMaxINR = 5000000 // ₹50,000
)

// ============================================================================
// VALIDATION CONSTANTS
// ============================================================================

// String Lengths
const (
	// MaxNameLength is the maximum length for names
	MaxNameLength = 100

	// MaxDescriptionLength is the maximum length for descriptions
	MaxDescriptionLength = 500

	// MaxAddressLength is the maximum length for address fields
	MaxAddressLength = 255

	// MaxPhoneLength is the maximum length for phone numbers
	MaxPhoneLength = 20

	// MaxEmailLength is the maximum length for email addresses
	MaxEmailLength = 255
)

// Numeric Limits
const (
	// MaxParcelWeightKg is the maximum parcel weight in kg
	MaxParcelWeightKg = 50.0

	// MaxParcelImages is the maximum number of parcel images
	MaxParcelImages = 5

	// MinRating is the minimum rating value
	MinRating = 1.0

	// MaxRating is the maximum rating value
	MaxRating = 5.0
)

// ============================================================================
// HTTP/API CONSTANTS
// ============================================================================

// Server Configuration
const (
	// ServerReadTimeout is the HTTP server read timeout
	ServerReadTimeout = 15 * time.Second

	// ServerWriteTimeout is the HTTP server write timeout
	ServerWriteTimeout = 15 * time.Second

	// ServerIdleTimeout is the HTTP server idle timeout
	ServerIdleTimeout = 60 * time.Second

	// ServerShutdownTimeout is the graceful shutdown timeout
	ServerShutdownTimeout = 30 * time.Second

	// MaxRequestBodySize is the maximum size of request body in bytes
	MaxRequestBodySize = 10 * 1024 * 1024 // 10 MB
)

// API Rate Limiting (per user)
const (
	// APIRateLimitPerMinute is the general API rate limit
	APIRateLimitPerMinute = 60

	// APIRateLimitWindow is the time window for API rate limiting
	APIRateLimitWindow = 1 * time.Minute

	// OrderCreationRateLimit is the rate limit for order creation
	OrderCreationRateLimit = 10

	// OrderCreationWindow is the time window for order creation rate limiting
	OrderCreationWindow = 1 * time.Hour
)

// ============================================================================
// NOTIFICATION CONSTANTS
// ============================================================================

// Notification Configuration
const (
	// NotificationRetryAttempts is the number of retry attempts for notifications
	NotificationRetryAttempts = 3

	// NotificationRetryDelay is the delay between notification retries
	NotificationRetryDelay = 1 * time.Second

	// NotificationBatchSize is the batch size for bulk notifications
	NotificationBatchSize = 100
)

// ============================================================================
// MONITORING CONSTANTS
// ============================================================================

// Metrics Collection
const (
	// MetricsCollectionInterval is how often metrics are collected
	MetricsCollectionInterval = 10 * time.Second

	// HealthCheckInterval is how often health checks are performed
	HealthCheckInterval = 30 * time.Second
)

// ============================================================================
// CURRENCY
// ============================================================================

const (
	// DefaultCurrency is the default currency code
	DefaultCurrency = "INR"

	// PaisePerRupee is the conversion factor from paise to rupees
	PaisePerRupee = 100
)
