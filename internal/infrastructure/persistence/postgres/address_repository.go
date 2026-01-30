package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"

	"gorm.io/gorm"
)

// AddressRepository implements domain.AddressRepository interface
type AddressRepository struct {
	db *gorm.DB
}

// NewAddressRepository creates a new AddressRepository
func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

// ============================================================================
// CREATE
// ============================================================================

// Create creates a new address
func (r *AddressRepository) Create(ctx context.Context, address *aggregates.Address) error {
	// Convert domain to database model
	model, err := DomainToAddress(address)
	if err != nil {
		return fmt.Errorf("failed to convert to database model: %w", err)
	}

	// Create in database
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create address: %w", err)
	}

	return nil
}

// ============================================================================
// READ
// ============================================================================

// FindByID finds an address by ID
func (r *AddressRepository) FindByID(ctx context.Context, id string) (*aggregates.Address, error) {
	var model AddressModel

	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find address: %w", err)
	}

	return AddressToDomain(&model)
}

// FindByUserID finds all addresses for a user
func (r *AddressRepository) FindByUserID(ctx context.Context, userID string) ([]*aggregates.Address, error) {
	var models []AddressModel

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("is_default DESC, last_used_at DESC NULLS LAST, created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find addresses: %w", err)
	}

	// Convert to domain models
	addresses := make([]*aggregates.Address, 0, len(models))
	for _, model := range models {
		address, err := AddressToDomain(&model)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// FindDefaultByUserID finds user's default address
func (r *AddressRepository) FindDefaultByUserID(ctx context.Context, userID string) (*aggregates.Address, error) {
	var model AddressModel

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ? AND deleted_at IS NULL", userID, true).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find default address: %w", err)
	}

	return AddressToDomain(&model)
}

// ============================================================================
// UPDATE
// ============================================================================

// Update updates an address (with optimistic locking)
func (r *AddressRepository) Update(ctx context.Context, address *aggregates.Address) error {
	// Convert domain to database model
	model, err := DomainToAddress(address)
	if err != nil {
		return fmt.Errorf("failed to convert to database model: %w", err)
	}

	// Update with optimistic locking
	result := r.db.WithContext(ctx).
		Model(&AddressModel{}).
		Where("id = ? AND version = ?", model.ID, model.Version-1).
		Updates(map[string]interface{}{
			"version":               model.Version,
			"label":                 model.Label,
			"address_line1":         model.AddressLine1,
			"address_line2":         model.AddressLine2,
			"landmark":              model.Landmark,
			"city":                  model.City,
			"state":                 model.State,
			"country":               model.Country,
			"postal_code":           model.PostalCode,
			"location":              gorm.Expr("ST_SetSRID(ST_GeomFromText(?), 4326)::geography", model.Location),
			"contact_name":          model.ContactName,
			"contact_phone":         model.ContactPhone,
			"is_default":            model.IsDefault,
			"is_active":             model.IsActive,
			"is_verified":           model.IsVerified,
			"usage_count":           model.UsageCount,
			"last_used_at":          model.LastUsedAt,
			"morning_usage_count":   model.MorningUsageCount,
			"afternoon_usage_count": model.AfternoonUsageCount,
			"evening_usage_count":   model.EveningUsageCount,
			"night_usage_count":     model.NightUsageCount,
			"weekday_usage_count":   model.WeekdayUsageCount,
			"weekend_usage_count":   model.WeekendUsageCount,
			"monthly_usage_count":   model.MonthlyUsageCount,
			"updated_at":            model.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update address: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrConcurrentModification
	}

	return nil
}

// UnsetDefault removes default flag from all user's addresses
func (r *AddressRepository) UnsetDefault(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Model(&AddressModel{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Update("is_default", false)

	if result.Error != nil {
		return fmt.Errorf("failed to unset default addresses: %w", result.Error)
	}

	return nil
}

// ============================================================================
// DELETE
// ============================================================================

// Delete soft deletes an address
func (r *AddressRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&AddressModel{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()"))

	if result.Error != nil {
		return fmt.Errorf("failed to delete address: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ============================================================================
// ANALYTICS
// ============================================================================

// IncrementUsage increments address usage statistics
// This is a lightweight operation optimized for frequent updates
func (r *AddressRepository) IncrementUsage(ctx context.Context, addressID string) error {
	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()
	month := int(now.Month())

	// Determine which time slot to increment
	var timeField string
	switch {
	case hour >= 6 && hour < 12:
		timeField = "morning_usage_count"
	case hour >= 12 && hour < 18:
		timeField = "afternoon_usage_count"
	case hour >= 18 && hour < 24:
		timeField = "evening_usage_count"
	default:
		timeField = "night_usage_count"
	}

	// Determine which day field to increment
	var dayField string
	if weekday >= time.Monday && weekday <= time.Friday {
		dayField = "weekday_usage_count"
	} else {
		dayField = "weekend_usage_count"
	}

	// Update using SQL to avoid race conditions
	result := r.db.WithContext(ctx).
		Model(&AddressModel{}).
		Where("id = ?", addressID).
		Updates(map[string]interface{}{
			"usage_count":         gorm.Expr("usage_count + 1"),
			timeField:             gorm.Expr(fmt.Sprintf("%s + 1", timeField)),
			dayField:              gorm.Expr(fmt.Sprintf("%s + 1", dayField)),
			"monthly_usage_count": gorm.Expr(fmt.Sprintf("jsonb_set(monthly_usage_count, '{%d}', (COALESCE((monthly_usage_count->>'%d')::int, 0) + 1)::text::jsonb)", month, month)),
			"last_used_at":        now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to increment usage: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ============================================================================
// SMART SUGGESTIONS
// ============================================================================

// FindSuggestedAddresses finds addresses that should be suggested at current time
// Uses ML-ready analytics to suggest addresses based on historical patterns
func (r *AddressRepository) FindSuggestedAddresses(ctx context.Context, userID string, limit int) ([]*aggregates.Address, error) {
	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()

	// Determine current time slot
	var timeWeight string
	switch {
	case hour >= 6 && hour < 12:
		timeWeight = "morning_usage_count"
	case hour >= 12 && hour < 18:
		timeWeight = "afternoon_usage_count"
	case hour >= 18 && hour < 24:
		timeWeight = "evening_usage_count"
	default:
		timeWeight = "night_usage_count"
	}

	// Determine day weight
	var dayWeight string
	if weekday >= time.Monday && weekday <= time.Friday {
		dayWeight = "weekday_usage_count"
	} else {
		dayWeight = "weekend_usage_count"
	}

	// Calculate suggestion score using weighted formula
	// Score = (time_match * 0.4) + (day_match * 0.3) + (recency * 0.3)
	query := fmt.Sprintf(`
		SELECT *,
			(
				(%s::float / NULLIF(usage_count, 0) * 0.4) +
				(%s::float / NULLIF(usage_count, 0) * 0.3) +
				(CASE 
					WHEN last_used_at IS NULL THEN 0
					WHEN last_used_at > NOW() - INTERVAL '1 day' THEN 0.3
					WHEN last_used_at > NOW() - INTERVAL '7 days' THEN 0.2
					WHEN last_used_at > NOW() - INTERVAL '30 days' THEN 0.1
					ELSE 0.05
				END)
			) as suggestion_score
		FROM addresses
		WHERE user_id = ?
		  AND deleted_at IS NULL
		  AND is_active = true
		  AND usage_count >= 2
		ORDER BY suggestion_score DESC, is_default DESC, last_used_at DESC NULLS LAST
		LIMIT ?
	`, timeWeight, dayWeight)

	var models []AddressModel
	err := r.db.WithContext(ctx).
		Raw(query, userID, limit).
		Scan(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find suggested addresses: %w", err)
	}

	// Convert to domain models
	addresses := make([]*aggregates.Address, 0, len(models))
	for _, model := range models {
		address, err := AddressToDomain(&model)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// FindNearest finds nearest addresses to given location
// Uses PostGIS for efficient spatial queries
func (r *AddressRepository) FindNearest(ctx context.Context, userID string, lat, lng float64, limit int) ([]*aggregates.Address, error) {
	var models []AddressModel

	// PostGIS query to find nearest addresses
	// Uses KNN operator <-> for efficient nearest neighbor search
	query := `
		SELECT *
		FROM addresses
		WHERE user_id = $1
		  AND deleted_at IS NULL
		  AND is_active = true
		ORDER BY location <-> ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
		LIMIT $4
	`

	err := r.db.WithContext(ctx).
		Raw(query, userID, lng, lat, limit).
		Scan(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find nearest addresses: %w", err)
	}

	// Convert to domain models
	addresses := make([]*aggregates.Address, 0, len(models))
	for _, model := range models {
		address, err := AddressToDomain(&model)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// ============================================================================
// BULK OPERATIONS
// ============================================================================

// FindByIDs finds multiple addresses by their IDs
func (r *AddressRepository) FindByIDs(ctx context.Context, ids []string) ([]*aggregates.Address, error) {
	var models []AddressModel

	err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find addresses by IDs: %w", err)
	}

	addresses := make([]*aggregates.Address, 0, len(models))
	for _, model := range models {
		address, err := AddressToDomain(&model)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// ============================================================================
// SEARCH
// ============================================================================

// SearchAddresses searches addresses by city, state, or postal code
func (r *AddressRepository) SearchAddresses(ctx context.Context, userID, query string, limit, offset int) ([]*aggregates.Address, int64, error) {
	var models []AddressModel
	var total int64

	// Build base query
	db := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID)

	// Add search criteria
	if query != "" {
		searchPattern := "%" + query + "%"
		db = db.Where(`
			city ILIKE ? OR 
			state ILIKE ? OR 
			postal_code ILIKE ? OR
			address_line1 ILIKE ?
		`, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	if err := db.Model(&AddressModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count addresses: %w", err)
	}

	// Get paginated results
	err := db.Limit(limit).Offset(offset).
		Order("is_default DESC, last_used_at DESC NULLS LAST").
		Find(&models).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to search addresses: %w", err)
	}

	// Convert to domain models
	addresses := make([]*aggregates.Address, 0, len(models))
	for _, model := range models {
		address, err := AddressToDomain(&model)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses, total, nil
}

// ============================================================================
// STATISTICS
// ============================================================================


func (r *AddressRepository) GetAddressStats(ctx context.Context, userID string) (*domain.AddressStats, error) {
	stats := &domain.AddressStats{}

	// Total addresses
	r.db.WithContext(ctx).
		Model(&AddressModel{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&stats.TotalAddresses)

	// Default address
	defaultAddr, err := r.FindDefaultByUserID(ctx, userID)
	if err == nil {
		stats.DefaultAddress = defaultAddr
	}

	// Most used address
	var mostUsedModel AddressModel
	err = r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("usage_count DESC").
		First(&mostUsedModel).Error

	if err == nil {
		mostUsed, _ := AddressToDomain(&mostUsedModel)
		stats.MostUsedAddress = mostUsed
	}

	// Recent addresses (last 5)
	var recentModels []AddressModel
	r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL AND last_used_at IS NOT NULL", userID).
		Order("last_used_at DESC").
		Limit(5).
		Find(&recentModels)

	stats.RecentAddresses = make([]*aggregates.Address, 0, len(recentModels))
	for _, model := range recentModels {
		addr, err := AddressToDomain(&model)
		if err == nil {
			stats.RecentAddresses = append(stats.RecentAddresses, addr)
		}
	}

	return stats, nil
}



func (r *AddressRepository) GetUsagePatterns(ctx context.Context, userID string) ([]domain.UsagePattern, error) {
	query := `
		SELECT 
			id as address_id,
			usage_count as total_usage,
			CASE WHEN usage_count > 0 THEN morning_usage_count::float / usage_count * 100 ELSE 0 END as morning_percentage,
			CASE WHEN usage_count > 0 THEN afternoon_usage_count::float / usage_count * 100 ELSE 0 END as afternoon_percentage,
			CASE WHEN usage_count > 0 THEN evening_usage_count::float / usage_count * 100 ELSE 0 END as evening_percentage,
			CASE WHEN usage_count > 0 THEN night_usage_count::float / usage_count * 100 ELSE 0 END as night_percentage,
			CASE WHEN (weekday_usage_count + weekend_usage_count) > 0 
				THEN weekday_usage_count::float / (weekday_usage_count + weekend_usage_count) * 100 
				ELSE 0 END as weekday_percentage,
			CASE WHEN (weekday_usage_count + weekend_usage_count) > 0 
				THEN weekend_usage_count::float / (weekday_usage_count + weekend_usage_count) * 100 
				ELSE 0 END as weekend_percentage,
			CASE WHEN usage_count > 1 AND last_used_at IS NOT NULL
				THEN EXTRACT(EPOCH FROM (last_used_at - created_at)) / 86400.0 / (usage_count - 1)
				ELSE 0 END as average_gap
		FROM addresses
		WHERE user_id = $1 AND deleted_at IS NULL AND usage_count > 0
		ORDER BY usage_count DESC
	`

	var patterns []domain.UsagePattern
	err := r.db.WithContext(ctx).Raw(query, userID).Scan(&patterns).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get usage patterns: %w", err)
	}

	return patterns, nil
}
