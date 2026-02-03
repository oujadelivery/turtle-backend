package valueobjects

import (
	"errors"
	"strings"
)

// OrderAddress represents pickup or delivery address for an order
// Immutable value object
type OrderAddress struct {
	addressLine1 string
	addressLine2 string
	landmark     string
	city         string
	state        string
	postalCode   string
	location     *Location
}

// NewOrderAddress creates a new order address
func NewOrderAddress(
	addressLine1, addressLine2, landmark, city, state, postalCode string,
	location *Location,
) (*OrderAddress, error) {
	// Validate required fields
	addressLine1 = strings.TrimSpace(addressLine1)
	if addressLine1 == "" {
		return nil, errors.New("address line 1 is required")
	}
	if len(addressLine1) > 255 {
		return nil, errors.New("address line 1 cannot exceed 255 characters")
	}

	city = strings.TrimSpace(city)
	if city == "" {
		return nil, errors.New("city is required")
	}
	if len(city) > 100 {
		return nil, errors.New("city cannot exceed 100 characters")
	}

	state = strings.TrimSpace(state)
	if state == "" {
		return nil, errors.New("state is required")
	}
	if len(state) > 100 {
		return nil, errors.New("state cannot exceed 100 characters")
	}

	if location == nil {
		return nil, errors.New("location coordinates are required")
	}

	// Trim optional fields
	addressLine2 = strings.TrimSpace(addressLine2)
	landmark = strings.TrimSpace(landmark)
	postalCode = strings.TrimSpace(postalCode)

	// Validate lengths
	if len(addressLine2) > 255 {
		return nil, errors.New("address line 2 cannot exceed 255 characters")
	}
	if len(landmark) > 255 {
		return nil, errors.New("landmark cannot exceed 255 characters")
	}
	if len(postalCode) > 20 {
		return nil, errors.New("postal code cannot exceed 20 characters")
	}

	return &OrderAddress{
		addressLine1: addressLine1,
		addressLine2: addressLine2,
		landmark:     landmark,
		city:         city,
		state:        state,
		postalCode:   postalCode,
		location:     location,
	}, nil
}

// Getters
func (oa OrderAddress) AddressLine1() string { return oa.addressLine1 }
func (oa OrderAddress) AddressLine2() string { return oa.addressLine2 }
func (oa OrderAddress) Landmark() string     { return oa.landmark }
func (oa OrderAddress) City() string         { return oa.city }
func (oa OrderAddress) State() string        { return oa.state }
func (oa OrderAddress) PostalCode() string   { return oa.postalCode }
func (oa OrderAddress) Location() *Location  { return oa.location }

// FormattedAddress returns human-readable address
func (oa OrderAddress) FormattedAddress() string {
	parts := []string{oa.addressLine1}
	
	if oa.addressLine2 != "" {
		parts = append(parts, oa.addressLine2)
	}
	if oa.landmark != "" {
		parts = append(parts, oa.landmark)
	}
	
	parts = append(parts, oa.city, oa.state)
	
	if oa.postalCode != "" {
		parts = append(parts, oa.postalCode)
	}
	
	return strings.Join(parts, ", ")
}

// ShortAddress returns a brief address (city, state)
func (oa OrderAddress) ShortAddress() string {
	return oa.city + ", " + oa.state
}

// HasPostalCode checks if postal code is provided
func (oa OrderAddress) HasPostalCode() bool {
	return oa.postalCode != ""
}

// HasLandmark checks if landmark is provided
func (oa OrderAddress) HasLandmark() bool {
	return oa.landmark != ""
}