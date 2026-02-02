package valueobjects

import (
	"errors"
	"strings"
)

// ParcelType represents the type of parcel
type ParcelType string

const (
	ParcelTypeDocument    ParcelType = "DOCUMENT"
	ParcelTypePackage     ParcelType = "PACKAGE"
	ParcelTypeFood        ParcelType = "FOOD"
	ParcelTypeFragile     ParcelType = "FRAGILE"
	ParcelTypeElectronics ParcelType = "ELECTRONICS"
	ParcelTypeClothing    ParcelType = "CLOTHING"
	ParcelTypeGroceries   ParcelType = "GROCERIES"
	ParcelTypeMedicines   ParcelType = "MEDICINES"
	ParcelTypeOther       ParcelType = "OTHER"
)

// Valid parcel types
var validParcelTypes = map[ParcelType]bool{
	ParcelTypeDocument:    true,
	ParcelTypePackage:     true,
	ParcelTypeFood:        true,
	ParcelTypeFragile:     true,
	ParcelTypeElectronics: true,
	ParcelTypeClothing:    true,
	ParcelTypeGroceries:   true,
	ParcelTypeMedicines:   true,
	ParcelTypeOther:       true,
}

// Parcel represents a parcel to be delivered
type Parcel struct {
	parcelType  ParcelType
	weight      float64 // in kg
	description string
	value       *Money   // declared value for insurance
	images      []string // URLs to parcel images
}

// NewParcel creates a new Parcel value object
func NewParcel(
	parcelType ParcelType,
	weight float64,
	description string,
	value *Money,
	images []string,
) (*Parcel, error) {
	// Validate parcel type
	if !validParcelTypes[parcelType] {
		return nil, errors.New("invalid parcel type")
	}

	// Validate weight (must be positive and reasonable)
	if weight <= 0 {
		return nil, errors.New("weight must be positive")
	}
	if weight > 50 { // Max 50kg
		return nil, errors.New("weight cannot exceed 50kg")
	}

	// Validate description
	description = strings.TrimSpace(description)
	if len(description) > 500 {
		return nil, errors.New("description cannot exceed 500 characters")
	}

	// Validate value
	if value != nil && value.IsNegative() {
		return nil, errors.New("parcel value cannot be negative")
	}

	// Validate images
	if len(images) > 5 {
		return nil, errors.New("cannot have more than 5 images")
	}

	return &Parcel{
		parcelType:  parcelType,
		weight:      weight,
		description: description,
		value:       value,
		images:      images,
	}, nil
}

// Type returns the parcel type
func (p Parcel) Type() ParcelType {
	return p.parcelType
}

// Weight returns the weight in kg
func (p Parcel) Weight() float64 {
	return p.weight
}

// Description returns the description
func (p Parcel) Description() string {
	return p.description
}

// Value returns the declared value
func (p Parcel) Value() *Money {
	return p.value
}

// Images returns the image URLs
func (p Parcel) Images() []string {
	return p.images
}

// RequiresSpecialHandling checks if parcel needs special care
func (p Parcel) RequiresSpecialHandling() bool {
	return p.parcelType == ParcelTypeFragile ||
		p.parcelType == ParcelTypeElectronics ||
		p.parcelType == ParcelTypeMedicines
}

// IsPerishable checks if parcel is perishable
func (p Parcel) IsPerishable() bool {
	return p.parcelType == ParcelTypeFood ||
		p.parcelType == ParcelTypeGroceries ||
		p.parcelType == ParcelTypeMedicines
}

// RequiresInsurance checks if parcel should be insured
func (p Parcel) RequiresInsurance() bool {
	if p.value == nil {
		return false
	}

	// Insure if value > 5000 (in major units)
	threshold := MustNewMoney(500000, p.value.Currency()) // 5000 in paise
	gt, _ := p.value.GreaterThan(*threshold)
	return gt
}

// CalculateInsuranceFee calculates insurance fee (1% of declared value)
func (p Parcel) CalculateInsuranceFee() *Money {
	if p.value == nil {
		return Zero("INR")
	}

	fee, _ := p.value.Multiply(0.01) // 1% of value
	return fee
}

// GetWeightCategory returns weight category for pricing
func (p Parcel) GetWeightCategory() string {
	switch {
	case p.weight <= 0.5:
		return "LIGHT" // Up to 500g
	case p.weight <= 2:
		return "MEDIUM" // Up to 2kg
	case p.weight <= 5:
		return "HEAVY" // Up to 5kg
	default:
		return "EXTRA_HEAVY" // Above 5kg
	}
}
