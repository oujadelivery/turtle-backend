package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"turtle/internal/domain"

	"gorm.io/gorm"
)

// RefreshTokenRepository implements domain.RefreshTokenRepository interface
type RefreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository
func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// ============================================================================
// CREATE
// ============================================================================

// Create creates a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, userID, token, device string, expiresAt time.Time) error {
	model := &RefreshTokenModel{
		UserID:    userID,
		Token:     token,
		Device:    &device,
		IsRevoked: false,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		// Check for unique constraint violation
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// ============================================================================
// READ
// ============================================================================

// FindByToken finds a refresh token
func (r *RefreshTokenRepository) FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	var model RefreshTokenModel

	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	// Convert to domain model
	device := ""
	if model.Device != nil {
		device = *model.Device
	}

	return &domain.RefreshToken{
		ID:         model.ID,
		UserID:     model.UserID,
		Token:      model.Token,
		Device:     device,
		DeviceInfo: model.DeviceInfo,
		IsRevoked:  model.IsRevoked,
		RevokedAt:  model.RevokedAt,
		CreatedAt:  model.CreatedAt,
		ExpiresAt:  model.ExpiresAt,
		LastUsedAt: model.LastUsedAt,
	}, nil
}

// FindByUserID finds all refresh tokens for a user
func (r *RefreshTokenRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error) {
	var models []RefreshTokenModel

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find refresh tokens: %w", err)
	}

	// Convert to domain models
	tokens := make([]*domain.RefreshToken, 0, len(models))
	for _, model := range models {
		device := ""
		if model.Device != nil {
			device = *model.Device
		}

		tokens = append(tokens, &domain.RefreshToken{
			ID:         model.ID,
			UserID:     model.UserID,
			Token:      model.Token,
			Device:     device,
			DeviceInfo: model.DeviceInfo,
			IsRevoked:  model.IsRevoked,
			RevokedAt:  model.RevokedAt,
			CreatedAt:  model.CreatedAt,
			ExpiresAt:  model.ExpiresAt,
			LastUsedAt: model.LastUsedAt,
		})
	}

	return tokens, nil
}

// FindActiveByUserID finds all active (non-revoked, non-expired) tokens for a user
func (r *RefreshTokenRepository) FindActiveByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error) {
	var models []RefreshTokenModel
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_revoked = ? AND expires_at > ?", userID, false, now).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find active refresh tokens: %w", err)
	}

	// Convert to domain models
	tokens := make([]*domain.RefreshToken, 0, len(models))
	for _, model := range models {
		device := ""
		if model.Device != nil {
			device = *model.Device
		}

		tokens = append(tokens, &domain.RefreshToken{
			ID:         model.ID,
			UserID:     model.UserID,
			Token:      model.Token,
			Device:     device,
			DeviceInfo: model.DeviceInfo,
			IsRevoked:  model.IsRevoked,
			RevokedAt:  model.RevokedAt,
			CreatedAt:  model.CreatedAt,
			ExpiresAt:  model.ExpiresAt,
			LastUsedAt: model.LastUsedAt,
		})
	}

	return tokens, nil
}

// ============================================================================
// UPDATE
// ============================================================================

// UpdateLastUsed updates the last used timestamp
func (r *RefreshTokenRepository) UpdateLastUsed(ctx context.Context, token string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("token = ?", token).
		Update("last_used_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to update last used: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Revoke revokes a refresh token
func (r *RefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("token = ?", token).
		Updates(map[string]interface{}{
			"is_revoked": true,
			"revoked_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to revoke token: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// RevokeAllForUser revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Updates(map[string]interface{}{
			"is_revoked": true,
			"revoked_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to revoke all tokens: %w", result.Error)
	}

	return nil
}

// RevokeAllForDevice revokes all tokens for a specific device
func (r *RefreshTokenRepository) RevokeAllForDevice(ctx context.Context, userID, device string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("user_id = ? AND device = ? AND is_revoked = ?", userID, device, false).
		Updates(map[string]interface{}{
			"is_revoked": true,
			"revoked_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to revoke device tokens: %w", result.Error)
	}

	return nil
}

// ============================================================================
// DELETE
// ============================================================================

// DeleteExpired deletes expired refresh tokens
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&RefreshTokenModel{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		fmt.Printf("âœ… Deleted %d expired refresh tokens\n", result.RowsAffected)
	}

	return nil
}

// DeleteRevoked deletes revoked tokens older than specified duration
func (r *RefreshTokenRepository) DeleteRevoked(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	result := r.db.WithContext(ctx).
		Where("is_revoked = ? AND revoked_at < ?", true, cutoff).
		Delete(&RefreshTokenModel{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete revoked tokens: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		fmt.Printf("âœ… Deleted %d revoked refresh tokens\n", result.RowsAffected)
	}

	return nil
}

// ============================================================================
// VALIDATION
// ============================================================================

// IsTokenValid checks if a token is valid (exists, not revoked, not expired)
func (r *RefreshTokenRepository) IsTokenValid(ctx context.Context, token string) (bool, error) {
	var count int64
	now := time.Now()

	err := r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("token = ? AND is_revoked = ? AND expires_at > ?", token, false, now).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to validate token: %w", err)
	}

	return count > 0, nil
}

// ============================================================================
// SESSION MANAGEMENT
// ============================================================================

// GetActiveSessions returns active sessions for a user grouped by device
type Session struct {
	Device     string
	LastUsed   time.Time
	CreatedAt  time.Time
	ExpiresAt  time.Time
	TokenCount int
}

func (r *RefreshTokenRepository) GetActiveSessions(ctx context.Context, userID string) ([]Session, error) {
	var sessions []Session
	now := time.Now()

	query := `
		SELECT 
			COALESCE(device, 'Unknown') as device,
			MAX(last_used_at) as last_used,
			MIN(created_at) as created_at,
			MAX(expires_at) as expires_at,
			COUNT(*) as token_count
		FROM refresh_tokens
		WHERE user_id = $1 
		  AND is_revoked = false 
		  AND expires_at > $2
		GROUP BY device
		ORDER BY last_used DESC NULLS LAST
	`

	err := r.db.WithContext(ctx).Raw(query, userID, now).Scan(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	return sessions, nil
}

// CountActiveTokens counts active tokens for a user
func (r *RefreshTokenRepository) CountActiveTokens(ctx context.Context, userID string) (int64, error) {
	var count int64
	now := time.Now()

	err := r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("user_id = ? AND is_revoked = ? AND expires_at > ?", userID, false, now).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count active tokens: %w", err)
	}

	return count, nil
}

// LimitActiveSessions limits the number of active sessions per user
// Revokes oldest sessions if limit is exceeded
func (r *RefreshTokenRepository) LimitActiveSessions(ctx context.Context, userID string, maxSessions int) error {
	// Get all active sessions ordered by last used
	tokens, err := r.FindActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// If under limit, nothing to do
	if len(tokens) <= maxSessions {
		return nil
	}

	// Revoke oldest sessions
	now := time.Now()
	tokensToRevoke := tokens[maxSessions:]

	tokenIDs := make([]uint, len(tokensToRevoke))
	for i, t := range tokensToRevoke {
		tokenIDs[i] = t.ID
	}

	result := r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("id IN ?", tokenIDs).
		Updates(map[string]interface{}{
			"is_revoked": true,
			"revoked_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to limit sessions: %w", result.Error)
	}

	fmt.Printf("âœ… Revoked %d old sessions for user %s\n", result.RowsAffected, userID)
	return nil
}

// ============================================================================
// STATISTICS
// ============================================================================

// GetTokenStats returns refresh token statistics
type TokenStats struct {
	TotalTokens     int64
	ActiveTokens    int64
	RevokedTokens   int64
	ExpiredTokens   int64
	RecentlyCreated int64 // Last 24 hours
	UniqueUsers     int64
	UniqueDevices   int64
}

func (r *RefreshTokenRepository) GetTokenStats(ctx context.Context) (*TokenStats, error) {
	stats := &TokenStats{}
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Total tokens
	r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Count(&stats.TotalTokens)

	// Active tokens
	r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("is_revoked = ? AND expires_at > ?", false, now).
		Count(&stats.ActiveTokens)

	// Revoked tokens
	r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("is_revoked = ?", true).
		Count(&stats.RevokedTokens)

	// Expired tokens (not revoked but past expiry)
	r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("is_revoked = ? AND expires_at < ?", false, now).
		Count(&stats.ExpiredTokens)

	// Recently created (last 24 hours)
	r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("created_at > ?", yesterday).
		Count(&stats.RecentlyCreated)

	// Unique users with active tokens
	r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("is_revoked = ? AND expires_at > ?", false, now).
		Distinct("user_id").
		Count(&stats.UniqueUsers)

	// Unique devices
	r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("device IS NOT NULL AND is_revoked = ? AND expires_at > ?", false, now).
		Distinct("device").
		Count(&stats.UniqueDevices)

	return stats, nil
}

// GetTokenStatsByDevice returns token statistics grouped by device
type TokenStatsByDevice struct {
	Device      string
	ActiveCount int64
	TotalCount  int64
	LastUsed    *time.Time
}

func (r *RefreshTokenRepository) GetTokenStatsByDevice(ctx context.Context) ([]TokenStatsByDevice, error) {
	var stats []TokenStatsByDevice
	now := time.Now()

	query := `
		SELECT 
			COALESCE(device, 'Unknown') as device,
			SUM(CASE WHEN is_revoked = false AND expires_at > $1 THEN 1 ELSE 0 END) as active_count,
			COUNT(*) as total_count,
			MAX(last_used_at) as last_used
		FROM refresh_tokens
		GROUP BY device
		ORDER BY active_count DESC
	`

	err := r.db.WithContext(ctx).Raw(query, now).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get token stats by device: %w", err)
	}

	return stats, nil
}

// ============================================================================
// CLEANUP
// ============================================================================

// CleanupOldTokens performs comprehensive cleanup
func (r *RefreshTokenRepository) CleanupOldTokens(ctx context.Context) (int64, error) {
	var totalDeleted int64
	now := time.Now()

	// Delete expired tokens
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&RefreshTokenModel{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", result.Error)
	}
	totalDeleted += result.RowsAffected

	// Delete revoked tokens older than 30 days
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	result = r.db.WithContext(ctx).
		Where("is_revoked = ? AND revoked_at < ?", true, thirtyDaysAgo).
		Delete(&RefreshTokenModel{})

	if result.Error != nil {
		return totalDeleted, fmt.Errorf("failed to delete old revoked tokens: %w", result.Error)
	}
	totalDeleted += result.RowsAffected

	fmt.Printf("âœ… Token Cleanup: Deleted %d old tokens\n", totalDeleted)
	return totalDeleted, nil
}
