package address

import (
	"errors"

	"turtle/db"
	"turtle/models"

	"gorm.io/gorm"
)

var (
	ErrAddressNotFound = errors.New("address not found")
	ErrUnauthorized    = errors.New("unauthorized access to address")
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

type CreateAddressInput struct {
	UserID       uint
	Label        string
	AddressLine1 string
	AddressLine2 string
	Landmark     string
	City         string
	State        string
	Country      string
	PostalCode   string
	Latitude     float64
	Longitude    float64
	ContactName  string
	ContactPhone string
	IsDefault    bool
}

type UpdateAddressInput struct {
	Label        *string
	AddressLine1 *string
	AddressLine2 *string
	Landmark     *string
	City         *string
	State        *string
	PostalCode   *string
	Latitude     *float64
	Longitude    *float64
	ContactName  *string
	ContactPhone *string
	IsActive     *bool
}

// CreateAddress creates a new address
func (s *Service) CreateAddress(input CreateAddressInput) (*models.Address, error) {
	// If this is set as default, remove default from other addresses
	if input.IsDefault {
		db.DB.Model(&models.Address{}).Where("user_id = ? AND is_default = true", input.UserID).Update("is_default", false)
	}

	address := &models.Address{
		UserID:       input.UserID,
		Label:        input.Label,
		AddressLine1: input.AddressLine1,
		AddressLine2: input.AddressLine2,
		Landmark:     input.Landmark,
		City:         input.City,
		State:        input.State,
		Country:      input.Country,
		PostalCode:   input.PostalCode,
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		ContactName:  input.ContactName,
		ContactPhone: input.ContactPhone,
		IsDefault:    input.IsDefault,
		IsActive:     true,
		IsVerified:   false,
	}

	if err := db.DB.Create(address).Error; err != nil {
		return nil, err
	}

	return address, nil
}

// GetAddressByID retrieves address by ID
func (s *Service) GetAddressByID(addressID uint) (*models.Address, error) {
	var address models.Address
	if err := db.DB.First(&address, addressID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAddressNotFound
		}
		return nil, err
	}
	return &address, nil
}

// GetUserAddresses retrieves all addresses for a user
func (s *Service) GetUserAddresses(userID uint) ([]*models.Address, error) {
	var addresses []*models.Address
	if err := db.DB.Where("user_id = ? AND is_active = true", userID).Order("is_default DESC, usage_count DESC").Find(&addresses).Error; err != nil {
		return nil, err
	}
	return addresses, nil
}

// GetDefaultAddress retrieves user's default address
func (s *Service) GetDefaultAddress(userID uint) (*models.Address, error) {
	var address models.Address
	if err := db.DB.Where("user_id = ? AND is_default = true AND is_active = true", userID).First(&address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return most used address if no default
			if err := db.DB.Where("user_id = ? AND is_active = true", userID).Order("usage_count DESC").First(&address).Error; err != nil {
				return nil, ErrAddressNotFound
			}
		} else {
			return nil, err
		}
	}
	return &address, nil
}

// GetSuggestedAddresses returns addresses based on usage patterns
func (s *Service) GetSuggestedAddresses(userID uint, timeOfDay string) ([]*models.Address, error) {
	var addresses []*models.Address
	query := db.DB.Where("user_id = ? AND is_active = true", userID)

	// Order by time-specific usage
	switch timeOfDay {
	case "MORNING":
		query = query.Order("morning_usage_count DESC")
	case "AFTERNOON":
		query = query.Order("afternoon_usage_count DESC")
	case "EVENING":
		query = query.Order("evening_usage_count DESC")
	case "NIGHT":
		query = query.Order("night_usage_count DESC")
	default:
		query = query.Order("usage_count DESC")
	}

	if err := query.Limit(5).Find(&addresses).Error; err != nil {
		return nil, err
	}

	return addresses, nil
}

// UpdateAddress updates an existing address
func (s *Service) UpdateAddress(addressID uint, userID uint, input UpdateAddressInput) (*models.Address, error) {
	address, err := s.GetAddressByID(addressID)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if address.UserID != userID {
		return nil, ErrUnauthorized
	}

	updates := make(map[string]interface{})
	if input.Label != nil {
		updates["label"] = *input.Label
	}
	if input.AddressLine1 != nil {
		updates["address_line1"] = *input.AddressLine1
	}
	if input.AddressLine2 != nil {
		updates["address_line2"] = *input.AddressLine2
	}
	if input.Landmark != nil {
		updates["landmark"] = *input.Landmark
	}
	if input.City != nil {
		updates["city"] = *input.City
	}
	if input.State != nil {
		updates["state"] = *input.State
	}
	if input.PostalCode != nil {
		updates["postal_code"] = *input.PostalCode
	}
	if input.Latitude != nil {
		updates["latitude"] = *input.Latitude
		updates["is_verified"] = false // Reset verification if coordinates change
	}
	if input.Longitude != nil {
		updates["longitude"] = *input.Longitude
		updates["is_verified"] = false
	}
	if input.ContactName != nil {
		updates["contact_name"] = *input.ContactName
	}
	if input.ContactPhone != nil {
		updates["contact_phone"] = *input.ContactPhone
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	if len(updates) > 0 {
		if err := db.DB.Model(address).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetAddressByID(addressID)
}

// DeleteAddress soft deletes an address
func (s *Service) DeleteAddress(addressID uint, userID uint) error {
	address, err := s.GetAddressByID(addressID)
	if err != nil {
		return err
	}

	if address.UserID != userID {
		return ErrUnauthorized
	}

	// Soft delete by marking as inactive
	if err := db.DB.Model(address).Update("is_active", false).Error; err != nil {
		return err
	}

	// If this was default, set another address as default
	if address.IsDefault {
		var nextAddress models.Address
		if err := db.DB.Where("user_id = ? AND is_active = true AND id != ?", userID, addressID).
			Order("usage_count DESC").First(&nextAddress).Error; err == nil {
			db.DB.Model(&nextAddress).Update("is_default", true)
		}
	}

	return nil
}

// SetDefaultAddress sets an address as default
func (s *Service) SetDefaultAddress(addressID uint, userID uint) (*models.Address, error) {
	address, err := s.GetAddressByID(addressID)
	if err != nil {
		return nil, err
	}

	if address.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Remove default from all other addresses
	db.DB.Model(&models.Address{}).Where("user_id = ? AND id != ?", userID, addressID).Update("is_default", false)

	// Set this as default
	if err := db.DB.Model(address).Update("is_default", true).Error; err != nil {
		return nil, err
	}

	return s.GetAddressByID(addressID)
}

// IncrementUsage updates address usage statistics
func (s *Service) IncrementUsage(addressID uint, orderID uint) error {
	address, err := s.GetAddressByID(addressID)
	if err != nil {
		return err
	}

	address.IncrementUsage()
	address.LastUsedForOrder = orderID

	return db.DB.Save(address).Error
}

// VerifyAddress marks address as verified
func (s *Service) VerifyAddress(addressID uint, userID uint) (*models.Address, error) {
	address, err := s.GetAddressByID(addressID)
	if err != nil {
		return nil, err
	}

	if address.UserID != userID {
		return nil, ErrUnauthorized
	}

	if err := db.DB.Model(address).Update("is_verified", true).Error; err != nil {
		return nil, err
	}

	return s.GetAddressByID(addressID)
}
