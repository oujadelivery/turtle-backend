package postgres

import (
	"context"
	"errors"
	"fmt"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRepository implements domain.UserRepository interface
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// ============================================================================
// CREATE
// ============================================================================

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *aggregates.User) error {
	// Convert domain to database model
	model, err := DomainToUser(user)
	if err != nil {
		return fmt.Errorf("failed to convert to database model: %w", err)
	}

	// Create in database with context
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		// Check for unique constraint violations
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Update domain aggregate with generated ID if needed
	// (ID is already set in this case)

	return nil
}

// ============================================================================
// READ
// ============================================================================

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id string) (*aggregates.User, error) {
	var model UserModel

	err := r.db.WithContext(ctx).
		Preload("CaptainProfile").
		Preload("AdminProfile").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return UserToDomain(&model)
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*aggregates.User, error) {
	var model UserModel

	err := r.db.WithContext(ctx).
		Preload("CaptainProfile").
		Preload("AdminProfile").
		Where("email = ? AND deleted_at IS NULL", email).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return UserToDomain(&model)
}

// FindByPhone finds a user by phone
func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (*aggregates.User, error) {
	var model UserModel

	err := r.db.WithContext(ctx).
		Preload("CaptainProfile").
		Preload("AdminProfile").
		Where("phone = ? AND deleted_at IS NULL", phone).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find user by phone: %w", err)
	}

	return UserToDomain(&model)
}

// FindByProviderID finds a user by provider ID (for social login)
func (r *UserRepository) FindByProviderID(ctx context.Context, provider, providerID string) (*aggregates.User, error) {
	var model UserModel

	err := r.db.WithContext(ctx).
		Preload("CaptainProfile").
		Preload("AdminProfile").
		Where("provider = ? AND provider_id = ? AND deleted_at IS NULL", provider, providerID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find user by provider: %w", err)
	}

	return UserToDomain(&model)
}

// ============================================================================
// UPDATE
// ============================================================================

// Update updates a user (with optimistic locking)
func (r *UserRepository) Update(ctx context.Context, user *aggregates.User) error {
	// Convert domain to database model
	model, err := DomainToUser(user)
	if err != nil {
		return fmt.Errorf("failed to convert to database model: %w", err)
	}

	// Update with optimistic locking
	// Only update if version matches (prevents concurrent modifications)
	result := r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("id = ? AND version = ?", model.ID, model.Version-1). // Version was incremented in domain
		Updates(map[string]interface{}{
			"version":          model.Version,
			"first_name":       model.FirstName,
			"last_name":        model.LastName,
			"profile_pic":      model.ProfilePic,
			"email":            model.Email,
			"email_verified":   model.EmailVerified,
			"phone":            model.Phone,
			"phone_verified":   model.PhoneVerified,
			"roles":            model.Roles,
			"status":           model.Status,
			"wallet_balance":   model.WalletBalance,
			"wallet_version":   model.WalletVersion,
			"total_orders":     model.TotalOrders,
			"total_deliveries": model.TotalDeliveries,
			"rating":           model.Rating,
			"total_ratings":    model.TotalRatings,
			"last_active_at":   model.LastActiveAt,
			"updated_at":       model.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update user: %w", result.Error)
	}

	// Check if any rows were affected (optimistic lock check)
	if result.RowsAffected == 0 {
		return domain.ErrConcurrentModification
	}

	// Update captain profile if exists
	if model.CaptainProfile != nil {
		if err := r.updateCaptainProfile(ctx, model.CaptainProfile); err != nil {
			return err
		}
	}

	// Update admin profile if exists
	if model.AdminProfile != nil {
		if err := r.updateAdminProfile(ctx, model.AdminProfile); err != nil {
			return err
		}
	}

	return nil
}

// updateCaptainProfile updates or creates captain profile
func (r *UserRepository) updateCaptainProfile(ctx context.Context, profile *CaptainProfileModel) error {
	// Use upsert (insert or update)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			UpdateAll: true,
		}).
		Create(profile).Error
}

// updateAdminProfile updates or creates admin profile
func (r *UserRepository) updateAdminProfile(ctx context.Context, profile *AdminProfileModel) error {
	// Use upsert (insert or update)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			UpdateAll: true,
		}).
		Create(profile).Error
}

// UpdateWallet updates user wallet balance (with optimistic locking)
// This is separated for better concurrency control on wallet operations
func (r *UserRepository) UpdateWallet(ctx context.Context, userID string, expectedVersion int64, newBalance int64) error {
	result := r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("id = ? AND wallet_version = ?", userID, expectedVersion).
		Updates(map[string]interface{}{
			"wallet_balance": newBalance,
			"wallet_version": expectedVersion + 1,
			"updated_at":     gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update wallet: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrConcurrentModification
	}

	return nil
}

// ============================================================================
// DELETE
// ============================================================================

// Delete soft deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()"))

	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ============================================================================
// GEOSPATIAL QUERIES (Captain-specific)
// ============================================================================

// FindCaptainsNearby finds available captains within radius
// Uses PostGIS for efficient geospatial queries
func (r *UserRepository) FindCaptainsNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*aggregates.User, error) {
	var models []UserModel

	// PostGIS query to find captains within radius
	// ST_DWithin uses geography type for accurate distance in meters
	// We use a subquery to get user IDs first, then join to load full user data
	query := `
		SELECT u.*
		FROM users u
		INNER JOIN captain_profiles cp ON u.id = cp.user_id
		WHERE u.deleted_at IS NULL
		  AND cp.is_available = true
		  AND cp.kyc_status = 'VERIFIED'
		  AND u.status = 'ACTIVE'
		  AND cp.current_location IS NOT NULL
		  AND ST_DWithin(
			cp.current_location,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3
		  )
		ORDER BY ST_Distance(
			cp.current_location,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
		) ASC
		LIMIT 20
	`

	err := r.db.WithContext(ctx).
		Preload("CaptainProfile").
		Raw(query, lng, lat, radiusKm*1000). // Convert km to meters
		Scan(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find nearby captains: %w", err)
	}

	// Convert to domain models
	users := make([]*aggregates.User, 0, len(models))
	for _, model := range models {
		user, err := UserToDomain(&model)
		if err != nil {
			continue // Skip invalid models
		}
		users = append(users, user)
	}

	return users, nil
}

// FindCaptainsByStatus finds captains by KYC status
func (r *UserRepository) FindCaptainsByStatus(ctx context.Context, status string) ([]*aggregates.User, error) {
	var models []UserModel

	err := r.db.WithContext(ctx).
		Joins("INNER JOIN captain_profiles ON users.id = captain_profiles.user_id").
		Preload("CaptainProfile").
		Where("captain_profiles.kyc_status = ? AND users.deleted_at IS NULL", status).
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find captains by status: %w", err)
	}

	// Convert to domain models
	users := make([]*aggregates.User, 0, len(models))
	for _, model := range models {
		user, err := UserToDomain(&model)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

// UpdateCaptainLocation updates captain's current location
// This is a lightweight operation that doesn't increment user version
func (r *UserRepository) UpdateCaptainLocation(ctx context.Context, captainID string, lat, lng float64) error {
	// Update captain profile location using PostGIS
	result := r.db.WithContext(ctx).
		Model(&CaptainProfileModel{}).
		Where("user_id = ?", captainID).
		Updates(map[string]interface{}{
			"current_location":    gorm.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography", lng, lat),
			"location_updated_at": gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update captain location: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ============================================================================
// SEARCH
// ============================================================================

// Search searches users by name, email, or phone
// Uses PostgreSQL full-text search with pg_trgm extension for fuzzy matching
func (r *UserRepository) Search(ctx context.Context, query string, role string, limit, offset int) ([]*aggregates.User, int64, error) {
	var models []UserModel
	var total int64

	// Build base query
	db := r.db.WithContext(ctx).
		Preload("CaptainProfile").
		Preload("AdminProfile").
		Where("deleted_at IS NULL")

	// Filter by role if specified
	if role != "" {
		db = db.Where("? = ANY(roles)", role)
	}

	// Search by name, email, or phone using trigram similarity
	if query != "" {
		searchPattern := "%" + query + "%"
		db = db.Where(`
			first_name ILIKE ? OR 
			last_name ILIKE ? OR 
			email ILIKE ? OR 
			phone ILIKE ?
		`, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	if err := db.Model(&UserModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated results
	err := db.Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}

	// Convert to domain models
	users := make([]*aggregates.User, 0, len(models))
	for _, model := range models {
		user, err := UserToDomain(&model)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, total, nil
}

// ============================================================================
// BULK OPERATIONS
// ============================================================================

// FindByIDs finds multiple users by their IDs
func (r *UserRepository) FindByIDs(ctx context.Context, ids []string) ([]*aggregates.User, error) {
	var models []UserModel

	err := r.db.WithContext(ctx).
		Preload("CaptainProfile").
		Preload("AdminProfile").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find users by IDs: %w", err)
	}

	users := make([]*aggregates.User, 0, len(models))
	for _, model := range models {
		user, err := UserToDomain(&model)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

// ============================================================================
// STATISTICS
// ============================================================================

// GetUserStats returns aggregated user statistics
type UserStats struct {
	TotalUsers       int64
	TotalCustomers   int64
	TotalCaptains    int64
	TotalAdmins      int64
	ActiveUsers      int64
	VerifiedCaptains int64
}

func (r *UserRepository) GetUserStats(ctx context.Context) (*UserStats, error) {
	var stats UserStats

	// Total users
	r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("deleted_at IS NULL").
		Count(&stats.TotalUsers)

	// Total customers
	r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("deleted_at IS NULL AND ? = ANY(roles)", "CUSTOMER").
		Count(&stats.TotalCustomers)

	// Total captains
	r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("deleted_at IS NULL AND ? = ANY(roles)", "CAPTAIN").
		Count(&stats.TotalCaptains)

	// Total admins
	r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("deleted_at IS NULL AND ? = ANY(roles)", "ADMIN").
		Count(&stats.TotalAdmins)

	// Active users (last 7 days)
	r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("deleted_at IS NULL AND last_active_at > NOW() - INTERVAL '7 days'").
		Count(&stats.ActiveUsers)

	// Verified captains
	r.db.WithContext(ctx).
		Model(&CaptainProfileModel{}).
		Where("kyc_status = 'VERIFIED' AND deleted_at IS NULL").
		Count(&stats.VerifiedCaptains)

	return &stats, nil
}
