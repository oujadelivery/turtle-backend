package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"turtle/internal/domain"

	"gorm.io/gorm"
)

// OTPRepository implements domain.OTPRepository interface
type OTPRepository struct {
	db *gorm.DB
}

// NewOTPRepository creates a new OTPRepository
func NewOTPRepository(db *gorm.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

// ============================================================================
// CREATE
// ============================================================================

// Create creates a new OTP session
func (r *OTPRepository) Create(ctx context.Context, target, code, purpose string, expiresAt time.Time) error {
	model := &OTPSessionModel{
		Target:    target,
		Code:      code,
		Purpose:   purpose,
		Used:      false,
		Attempts:  0,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create OTP session: %w", err)
	}

	return nil
}

// ============================================================================
// READ
// ============================================================================

// FindByTarget finds the latest OTP session for target and purpose
func (r *OTPRepository) FindByTarget(ctx context.Context, target, purpose string) (*domain.OTPSession, error) {
	var model OTPSessionModel

	// Find the most recent unused OTP for this target and purpose
	err := r.db.WithContext(ctx).
		Where("target = ? AND purpose = ? AND used = ?", target, purpose, false).
		Order("created_at DESC").
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find OTP session: %w", err)
	}

	// Convert to domain model
	return &domain.OTPSession{
		ID:        model.ID,
		Target:    model.Target,
		Code:      model.Code,
		Purpose:   model.Purpose,
		Used:      model.Used,
		Attempts:  model.Attempts,
		CreatedAt: model.CreatedAt,
		ExpiresAt: model.ExpiresAt,
	}, nil
}

// ============================================================================
// UPDATE
// ============================================================================

// MarkAsUsed marks an OTP session as used
func (r *OTPRepository) MarkAsUsed(ctx context.Context, target, purpose string) error {
	result := r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("target = ? AND purpose = ? AND used = ?", target, purpose, false).
		Update("used", true)

	if result.Error != nil {
		return fmt.Errorf("failed to mark OTP as used: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// IncrementAttempts increments failed verification attempts
func (r *OTPRepository) IncrementAttempts(ctx context.Context, target, purpose string) error {
	result := r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("target = ? AND purpose = ? AND used = ?", target, purpose, false).
		Update("attempts", gorm.Expr("attempts + 1"))

	if result.Error != nil {
		return fmt.Errorf("failed to increment attempts: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ============================================================================
// DELETE
// ============================================================================

// DeleteExpired deletes expired OTP sessions
// This should be called periodically by a cleanup job
func (r *OTPRepository) DeleteExpired(ctx context.Context) error {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&OTPSessionModel{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete expired OTP sessions: %w", result.Error)
	}

	// Log how many were deleted
	if result.RowsAffected > 0 {
		fmt.Printf("âœ… Deleted %d expired OTP sessions\n", result.RowsAffected)
	}

	return nil
}

// DeleteUsed deletes used OTP sessions older than specified duration
func (r *OTPRepository) DeleteUsed(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	result := r.db.WithContext(ctx).
		Where("used = ? AND created_at < ?", true, cutoff).
		Delete(&OTPSessionModel{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete used OTP sessions: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		fmt.Printf("âœ… Deleted %d used OTP sessions\n", result.RowsAffected)
	}

	return nil
}

// ============================================================================
// BULK OPERATIONS
// ============================================================================

// FindActiveByTarget finds all active OTP sessions for a target
func (r *OTPRepository) FindActiveByTarget(ctx context.Context, target string) ([]*domain.OTPSession, error) {
	var models []OTPSessionModel

	err := r.db.WithContext(ctx).
		Where("target = ? AND used = ? AND expires_at > ?", target, false, time.Now()).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find active OTP sessions: %w", err)
	}

	// Convert to domain models
	sessions := make([]*domain.OTPSession, 0, len(models))
	for _, model := range models {
		sessions = append(sessions, &domain.OTPSession{
			ID:        model.ID,
			Target:    model.Target,
			Code:      model.Code,
			Purpose:   model.Purpose,
			Used:      model.Used,
			Attempts:  model.Attempts,
			CreatedAt: model.CreatedAt,
			ExpiresAt: model.ExpiresAt,
		})
	}

	return sessions, nil
}

// RevokeAll revokes all OTP sessions for a target
// Useful when user logs out or changes phone/email
func (r *OTPRepository) RevokeAll(ctx context.Context, target string) error {
	result := r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("target = ? AND used = ?", target, false).
		Update("used", true)

	if result.Error != nil {
		return fmt.Errorf("failed to revoke OTP sessions: %w", result.Error)
	}

	return nil
}

// ============================================================================
// RATE LIMITING HELPERS
// ============================================================================

// CountRecentOTPRequests counts OTP requests in the last time window
// Used for rate limiting OTP generation
func (r *OTPRepository) CountRecentOTPRequests(ctx context.Context, target string, window time.Duration) (int64, error) {
	var count int64
	cutoff := time.Now().Add(-window)

	err := r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("target = ? AND created_at > ?", target, cutoff).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count OTP requests: %w", err)
	}

	return count, nil
}

// GetLastOTPTime gets the timestamp of the last OTP sent to target
func (r *OTPRepository) GetLastOTPTime(ctx context.Context, target string) (*time.Time, error) {
	var model OTPSessionModel

	err := r.db.WithContext(ctx).
		Where("target = ?", target).
		Order("created_at DESC").
		Select("created_at").
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No OTP sent yet
		}
		return nil, fmt.Errorf("failed to get last OTP time: %w", err)
	}

	return &model.CreatedAt, nil
}

// ============================================================================
// STATISTICS
// ============================================================================

// GetOTPStats returns OTP usage statistics
type OTPStats struct {
	TotalSent      int64
	TotalUsed      int64
	TotalExpired   int64
	TotalFailed    int64 // Attempts >= 5
	ActiveSessions int64
	RecentRequests int64 // Last 24 hours
}

func (r *OTPRepository) GetOTPStats(ctx context.Context) (*OTPStats, error) {
	stats := &OTPStats{}
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Total sent
	r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Count(&stats.TotalSent)

	// Total used
	r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("used = ?", true).
		Count(&stats.TotalUsed)

	// Total expired (unused and past expiry)
	r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("used = ? AND expires_at < ?", false, now).
		Count(&stats.TotalExpired)

	// Total failed (max attempts reached)
	r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("used = ? AND attempts >= 5", false).
		Count(&stats.TotalFailed)

	// Active sessions (unused and not expired)
	r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("used = ? AND expires_at > ?", false, now).
		Count(&stats.ActiveSessions)

	// Recent requests (last 24 hours)
	r.db.WithContext(ctx).
		Model(&OTPSessionModel{}).
		Where("created_at > ?", yesterday).
		Count(&stats.RecentRequests)

	return stats, nil
}

// GetOTPStatsByPurpose returns OTP statistics grouped by purpose
type OTPStatsByPurpose struct {
	Purpose     string
	TotalSent   int64
	TotalUsed   int64
	SuccessRate float64
}

func (r *OTPRepository) GetOTPStatsByPurpose(ctx context.Context) ([]OTPStatsByPurpose, error) {
	var stats []OTPStatsByPurpose

	query := `
		SELECT 
			purpose,
			COUNT(*) as total_sent,
			SUM(CASE WHEN used = true THEN 1 ELSE 0 END) as total_used,
			CASE WHEN COUNT(*) > 0 
				THEN (SUM(CASE WHEN used = true THEN 1 ELSE 0 END)::float / COUNT(*) * 100)
				ELSE 0 
			END as success_rate
		FROM otp_sessions
		GROUP BY purpose
		ORDER BY total_sent DESC
	`

	err := r.db.WithContext(ctx).Raw(query).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get OTP stats by purpose: %w", err)
	}

	return stats, nil
}

// ============================================================================
// CLEANUP OPERATIONS
// ============================================================================

// CleanupOldSessions performs comprehensive cleanup of old OTP sessions
// This should be run periodically (e.g., daily via cron job)
func (r *OTPRepository) CleanupOldSessions(ctx context.Context) (int64, error) {
	// Delete sessions that are:
	// 1. Expired
	// 2. Used and older than 7 days
	// 3. Failed (5+ attempts) and older than 1 day

	var totalDeleted int64
	now := time.Now()

	// Delete expired sessions
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&OTPSessionModel{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", result.Error)
	}
	totalDeleted += result.RowsAffected

	// Delete used sessions older than 7 days
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	result = r.db.WithContext(ctx).
		Where("used = ? AND created_at < ?", true, sevenDaysAgo).
		Delete(&OTPSessionModel{})

	if result.Error != nil {
		return totalDeleted, fmt.Errorf("failed to delete used sessions: %w", result.Error)
	}
	totalDeleted += result.RowsAffected

	// Delete failed sessions older than 1 day
	oneDayAgo := now.Add(-24 * time.Hour)
	result = r.db.WithContext(ctx).
		Where("attempts >= 5 AND created_at < ?", oneDayAgo).
		Delete(&OTPSessionModel{})

	if result.Error != nil {
		return totalDeleted, fmt.Errorf("failed to delete failed sessions: %w", result.Error)
	}
	totalDeleted += result.RowsAffected

	fmt.Printf("âœ… OTP Cleanup: Deleted %d old sessions\n", totalDeleted)
	return totalDeleted, nil
}

// ============================================================================
// VALIDATION HELPERS
// ============================================================================

// IsOTPValid checks if an OTP session is valid for verification
func (r *OTPRepository) IsOTPValid(ctx context.Context, target, purpose, code string) (bool, error) {
	session, err := r.FindByTarget(ctx, target, purpose)
	if err != nil {
		return false, err
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return false, fmt.Errorf("OTP has expired")
	}

	// Check if already used
	if session.Used {
		return false, fmt.Errorf("OTP already used")
	}

	// Check attempts
	if session.Attempts >= 5 {
		return false, fmt.Errorf("maximum attempts exceeded")
	}

	// Check code
	if session.Code != code {
		return false, fmt.Errorf("invalid OTP code")
	}

	return true, nil
}
