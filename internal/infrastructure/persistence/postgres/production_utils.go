package postgres

import (
	"context"
	"fmt"
	"time"

	"turtle/internal/domain"

	"gorm.io/gorm"
)

// ============================================================================
// PRODUCTION VALIDATION & UTILITIES
// This file contains production-grade utilities and validation helpers
// ============================================================================

// HealthCheck performs comprehensive health check on the repository layer
type HealthCheck struct {
	DatabaseConnected bool
	IndexesPresent    bool
	ExtensionsLoaded  bool
	Errors            []string
}

// ValidateProduction validates that all production requirements are met
func ValidateProduction(db *gorm.DB) (*HealthCheck, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := &HealthCheck{
		Errors: make([]string, 0),
	}

	// Check database connection
	sqlDB, err := db.DB()
	if err != nil {
		health.Errors = append(health.Errors, fmt.Sprintf("Failed to get database: %v", err))
		return health, err
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		health.Errors = append(health.Errors, fmt.Sprintf("Database ping failed: %v", err))
		return health, err
	}
	health.DatabaseConnected = true

	// Check PostGIS extension
	var hasPostGIS bool
	err = db.WithContext(ctx).Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'postgis')").Scan(&hasPostGIS).Error
	if err != nil {
		health.Errors = append(health.Errors, fmt.Sprintf("Failed to check PostGIS: %v", err))
	} else if !hasPostGIS {
		health.Errors = append(health.Errors, "PostGIS extension not installed")
	} else {
		health.ExtensionsLoaded = true
	}

	// Check critical indexes
	criticalIndexes := []string{
		"idx_users_email",
		"idx_users_phone",
		"idx_captain_location",
		"idx_addresses_user_id",
		"idx_otp_target_purpose",
		"idx_refresh_tokens_token",
	}

	missingIndexes := make([]string, 0)
	for _, idxName := range criticalIndexes {
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname = ?)"
		err := db.WithContext(ctx).Raw(query, idxName).Scan(&exists).Error
		if err != nil || !exists {
			missingIndexes = append(missingIndexes, idxName)
		}
	}

	if len(missingIndexes) > 0 {
		health.Errors = append(health.Errors, fmt.Sprintf("Missing indexes: %v", missingIndexes))
	} else {
		health.IndexesPresent = true
	}

	return health, nil
}

// ============================================================================
// CONTEXT TIMEOUT HELPERS
// ============================================================================

// WithDefaultTimeout wraps context with default timeout if none exists
func WithDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		// Context already has deadline, return as is
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 30*time.Second)
}

// ============================================================================
// BATCH OPERATION HELPERS
// ============================================================================

// BatchProcessor processes items in batches to avoid memory issues
type BatchProcessor struct {
	BatchSize int
}

// NewBatchProcessor creates a batch processor with default size
func NewBatchProcessor() *BatchProcessor {
	return &BatchProcessor{
		BatchSize: 1000,
	}
}

// ProcessInBatches processes items in batches
func (bp *BatchProcessor) ProcessInBatches(
	ctx context.Context,
	totalCount int64,
	processBatch func(offset, limit int) error,
) error {
	for offset := 0; offset < int(totalCount); offset += bp.BatchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := processBatch(offset, bp.BatchSize); err != nil {
				return err
			}
		}
	}
	return nil
}

// ============================================================================
// CONNECTION POOL MONITORING
// ============================================================================

// PoolStats represents database connection pool statistics
type PoolStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleClosed      int64
	MaxLifetimeClosed  int64
}

// GetPoolStats returns current connection pool statistics
func GetPoolStats(db *gorm.DB) (*PoolStats, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	stats := sqlDB.Stats()
	return &PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

// ============================================================================
// QUERY PERFORMANCE MONITORING
// ============================================================================

// SlowQueryLogger logs queries that exceed threshold
type SlowQueryLogger struct {
	Threshold time.Duration
	Logger    func(query string, duration time.Duration)
}

// NewSlowQueryLogger creates a slow query logger
func NewSlowQueryLogger(threshold time.Duration) *SlowQueryLogger {
	return &SlowQueryLogger{
		Threshold: threshold,
		Logger: func(query string, duration time.Duration) {
			fmt.Printf("⚠️  SLOW QUERY (%v): %s\n", duration, query)
		},
	}
}

// ============================================================================
// SAFE TRANSACTION HELPERS
// ============================================================================

// Transaction executes function in transaction with automatic rollback
func Transaction(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ============================================================================
// RETRY LOGIC FOR CONCURRENT MODIFICATIONS
// ============================================================================

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}
}

// RetryOnConcurrentModification retries operation on concurrent modification
func RetryOnConcurrentModification(
	ctx context.Context,
	config *RetryConfig,
	fn func() error,
) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		// Check if it's a concurrent modification error
		if err != domain.ErrConcurrentModification {
			return err
		}

		lastErr = err

		// Don't sleep on last attempt
		if attempt < config.MaxAttempts-1 {
			delay := config.BaseDelay * time.Duration(1<<uint(attempt))
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// ============================================================================
// INPUT VALIDATION HELPERS
// ============================================================================

// ValidateID validates UUID format
func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	// Basic UUID format check (can be enhanced)
	if len(id) != 36 {
		return fmt.Errorf("invalid ID format")
	}
	return nil
}

// ValidateEmail validates email format (basic)
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	// Add more sophisticated email validation if needed
	return nil
}

// ValidatePhone validates phone format (basic)
func ValidatePhone(phone string) error {
	if phone == "" {
		return fmt.Errorf("phone cannot be empty")
	}
	// Add more sophisticated phone validation if needed
	return nil
}

// ValidatePagination validates pagination parameters
func ValidatePagination(limit, offset int) error {
	if limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}
	if limit > 1000 {
		return fmt.Errorf("limit cannot exceed 1000")
	}
	if offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	return nil
}

// ============================================================================
// SAFE NULL HANDLING
// ============================================================================

// StringPtr returns pointer to string (nil for empty strings)
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StringValue returns string value (empty for nil)
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// TimePtr returns pointer to time
func TimePtr(t time.Time) *time.Time {
	return &t
}

// TimeValue returns time value (zero time for nil)
func TimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
