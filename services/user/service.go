package user

import (
	"errors"
	"time"

	"turtle/db"
	"turtle/graph/model"
	"turtle/models"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUnauthorized = errors.New("unauthorized")
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GetUserByID retrieves user by ID
func (s *Service) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := db.DB.Preload("Addresses").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateProfile updates user profile
func (s *Service) UpdateProfile(userID uint, firstName, lastName, email, phone *string) (*models.User, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if firstName != nil {
		updates["first_name"] = *firstName
	}
	if lastName != nil {
		updates["last_name"] = *lastName
	}
	if email != nil {
		// Check if email already exists
		var existingUser models.User
		if err := db.DB.Where("email = ? AND id != ?", *email, userID).First(&existingUser).Error; err == nil {
			return nil, errors.New("email already in use")
		}
		updates["email"] = *email
		updates["email_verified"] = false
	}
	if phone != nil {
		// Check if phone already exists
		var existingUser models.User
		if err := db.DB.Where("phone = ? AND id != ?", *phone, userID).First(&existingUser).Error; err == nil {
			return nil, errors.New("phone already in use")
		}
		updates["phone"] = *phone
		updates["phone_verified"] = false
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetUserByID(userID)
}

// UpdateProfilePicture updates user profile picture
func (s *Service) UpdateProfilePicture(userID uint, imageURL string) (*models.User, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if err := db.DB.Model(&user).Update("profile_pic", imageURL).Error; err != nil {
		return nil, err
	}

	return s.GetUserByID(userID)
}

// UpdateCaptainProfile updates captain-specific profile
func (s *Service) UpdateCaptainProfile(userID uint, vehicleType *model.VehicleType, vehicleNumber, vehicleModel, licenseNumber *string, licenseExpiry *time.Time) (*models.User, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if !user.IsCaptain() {
		return nil, errors.New("user is not a captain")
	}

	updates := make(map[string]interface{})
	if vehicleType != nil {
		updates["vehicle_type"] = *vehicleType
	}
	if vehicleNumber != nil {
		updates["vehicle_number"] = *vehicleNumber
	}
	if vehicleModel != nil {
		updates["vehicle_model"] = *vehicleModel
	}
	if licenseNumber != nil {
		updates["license_number"] = *licenseNumber
	}
	if licenseExpiry != nil {
		updates["license_expiry"] = *licenseExpiry
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetUserByID(userID)
}

// ToggleAvailability toggles captain availability
func (s *Service) ToggleAvailability(userID uint, isAvailable bool) (*models.User, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if !user.IsCaptain() {
		return nil, errors.New("user is not a captain")
	}

	if err := db.DB.Model(&user).Update("is_available", isAvailable).Error; err != nil {
		return nil, err
	}

	return s.GetUserByID(userID)
}

// UpdateCurrentLocation updates captain's current location
func (s *Service) UpdateCurrentLocation(userID uint, lat, lng float64) (*models.User, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"current_lat":                 lat,
		"current_lng":                 lng,
		"current_location_updated_at": now,
		"last_active_at":              now,
	}

	if err := db.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Also update in CurrentLocation table
	var currentLoc models.CurrentLocation
	if err := db.DB.Where("user_id = ?", userID).First(&currentLoc).Error; err != nil {
		// Create new record
		currentLoc = models.CurrentLocation{
			UserID:    userID,
			Latitude:  lat,
			Longitude: lng,
			Source:    "GPS",
		}
		db.DB.Create(&currentLoc)
	} else {
		// Update existing
		db.DB.Model(&currentLoc).Updates(map[string]interface{}{
			"latitude":  lat,
			"longitude": lng,
			"source":    "GPS",
		})
	}

	return s.GetUserByID(userID)
}

// GetNearbyCaptains finds available captains near a location
func (s *Service) GetNearbyCaptains(lat, lng, radiusKm float64) ([]*models.User, error) {
	var captains []*models.User

	// Using Haversine formula to find nearby captains
	// Note: For production, use PostGIS or spatial indexing
	query := `
        SELECT * FROM users 
        WHERE role = 'CAPTAIN' 
        AND is_available = true 
        AND status = 'ACTIVE'
        AND kyc_status = 'VERIFIED'
        AND (
            6371 * acos(
                cos(radians(?)) * cos(radians(current_lat)) *
                cos(radians(current_lng) - radians(?)) +
                sin(radians(?)) * sin(radians(current_lat))
            )
        ) <= ?
        ORDER BY (
            6371 * acos(
                cos(radians(?)) * cos(radians(current_lat)) *
                cos(radians(current_lng) - radians(?)) +
                sin(radians(?)) * sin(radians(current_lat))
            )
        ) ASC
        LIMIT 20
    `

	if err := db.DB.Raw(query, lat, lng, lat, radiusKm, lat, lng, lat).Scan(&captains).Error; err != nil {
		return nil, err
	}

	return captains, nil
}

// SearchUsers searches users (admin only)
func (s *Service) SearchUsers(query, role string, page, pageSize int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	dbQuery := db.DB.Model(&models.User{})

	if query != "" {
		dbQuery = dbQuery.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ? OR phone ILIKE ?",
			"%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%")
	}

	if role != "" {
		dbQuery = dbQuery.Where("role = ?", role)
	}

	dbQuery.Count(&total)

	offset := (page - 1) * pageSize
	if err := dbQuery.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateUserStatus updates user status (admin only)
func (s *Service) UpdateUserStatus(userID uint, status string) (*models.User, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if err := db.DB.Model(&user).Update("status", status).Error; err != nil {
		return nil, err
	}

	return s.GetUserByID(userID)
}

// ApproveKYC approves or rejects captain KYC (admin only)
func (s *Service) ApproveKYC(userID uint, approved bool, reason *string) (*models.User, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if !user.IsCaptain() {
		return nil, errors.New("user is not a captain")
	}

	updates := make(map[string]interface{})
	if approved {
		updates["kyc_status"] = "VERIFIED"
		now := time.Now()
		updates["onboarding_completed_at"] = now
	} else {
		updates["kyc_status"] = "REJECTED"
	}

	if err := db.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	// TODO: Send notification to captain

	return s.GetUserByID(userID)
}

// CalculateUserStats recalculates user statistics
func (s *Service) CalculateUserStats(userID uint) error {
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		return err
	}

	// Calculate rating
	var avgRating float64
	var totalRatings int64
	db.DB.Model(&models.Rating{}).
		Where("rated_user_id = ?", userID).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as total").
		Row().Scan(&avgRating, &totalRatings)

	updates := map[string]interface{}{
		"rating":        avgRating,
		"total_ratings": totalRatings,
	}

	if user.IsCaptain() {
		// Calculate captain-specific stats
		var completedOrders, totalOrders int64
		db.DB.Model(&models.Order{}).Where("captain_id = ? AND status = 'DELIVERED'", userID).Count(&completedOrders)
		db.DB.Model(&models.Order{}).Where("captain_id = ?", userID).Count(&totalOrders)

		updates["total_deliveries"] = completedOrders

		var completionRate float64
		if totalOrders > 0 {
			completionRate = float64(completedOrders) / float64(totalOrders) * 100
		}
		updates["completion_rate"] = completionRate
	} else {
		// Calculate customer stats
		var totalOrders int64
		db.DB.Model(&models.Order{}).Where("customer_id = ?", userID).Count(&totalOrders)
		updates["total_orders"] = totalOrders
	}

	return db.DB.Model(&user).Updates(updates).Error
}
