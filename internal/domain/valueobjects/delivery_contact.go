package valueobjects

import (
	"errors"
	"regexp"
	"strings"
)

// DeliveryContact represents sender or receiver in an order
// This is NOT a User - it's a value object for delivery participants
type DeliveryContact struct {
	name           string
	phone          string
	alternatePhone *string
	notes          string // Special instructions (e.g., "ring bell twice")
}

// NewDeliveryContact creates a new delivery contact
func NewDeliveryContact(name, phone string, alternatePhone *string, notes string) (*DeliveryContact, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("contact name is required")
	}
	if len(name) > 100 {
		return nil, errors.New("name cannot exceed 100 characters")
	}

	// Validate phone
	phone = strings.TrimSpace(phone)
	if !isValidPhone(phone) {
		return nil, errors.New("invalid phone number format")
	}

	// Validate alternate phone if provided
	if alternatePhone != nil {
		alt := strings.TrimSpace(*alternatePhone)
		if alt != "" {
			if !isValidPhone(alt) {
				return nil, errors.New("invalid alternate phone number format")
			}
			alternatePhone = &alt
		} else {
			alternatePhone = nil // Empty string becomes nil
		}
	}

	// Validate notes
	notes = strings.TrimSpace(notes)
	if len(notes) > 500 {
		return nil, errors.New("notes cannot exceed 500 characters")
	}

	return &DeliveryContact{
		name:           name,
		phone:          phone,
		alternatePhone: alternatePhone,
		notes:          notes,
	}, nil
}

// Getters
func (dc DeliveryContact) Name() string           { return dc.name }
func (dc DeliveryContact) Phone() string          { return dc.phone }
func (dc DeliveryContact) AlternatePhone() *string { return dc.alternatePhone }
func (dc DeliveryContact) Notes() string          { return dc.notes }

// HasAlternatePhone checks if alternate phone is provided
func (dc DeliveryContact) HasAlternatePhone() bool {
	return dc.alternatePhone != nil && *dc.alternatePhone != ""
}

// isValidPhone validates phone number format
// Accepts international format: +[country code][number]
// Examples: +919876543210, +12025551234
func isValidPhone(phone string) bool {
	// Basic phone validation - E.164 format
	// Must start with +, followed by 1-3 digit country code, then 4-14 digits
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	return phoneRegex.MatchString(phone)
}