package valueobjects

import (
	"errors"
	"regexp"
	"strings"
)

// ContactInfo represents contact information for order participants
// Used for pickup person and delivery recipient who may not be app users
type ContactInfo struct {
	name  string
	phone string
}

var (
	// Phone validation regex - supports various formats
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{9,14}$`)
)

// NewContactInfo creates a new ContactInfo value object
func NewContactInfo(name, phone string) (*ContactInfo, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	if len(name) < 2 {
		return nil, errors.New("name must be at least 2 characters")
	}
	if len(name) > 100 {
		return nil, errors.New("name cannot exceed 100 characters")
	}

	// Validate phone
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	if phone == "" {
		return nil, errors.New("phone number is required")
	}

	// Basic phone validation
	if !phoneRegex.MatchString(phone) {
		return nil, errors.New("invalid phone number format")
	}

	return &ContactInfo{
		name:  name,
		phone: phone,
	}, nil
}

// Name returns the contact name
func (c ContactInfo) Name() string {
	return c.name
}

// Phone returns the phone number
func (c ContactInfo) Phone() string {
	return c.phone
}

// FormattedPhone returns phone in a readable format
// Example: +919876543210 -> +91 98765 43210
func (c ContactInfo) FormattedPhone() string {
	if len(c.phone) <= 10 {
		return c.phone
	}

	// Basic formatting for Indian numbers
	if strings.HasPrefix(c.phone, "+91") && len(c.phone) == 13 {
		return c.phone[:3] + " " + c.phone[3:8] + " " + c.phone[8:]
	}

	return c.phone
}

// Equals checks if two ContactInfo are equal
func (c ContactInfo) Equals(other ContactInfo) bool {
	return c.name == other.name && c.phone == other.phone
}

// String returns string representation
func (c ContactInfo) String() string {
	return c.name + " (" + c.FormattedPhone() + ")"
}

// Validate additional checks (can be extended)
func (c ContactInfo) Validate() error {
	if c.name == "" {
		return errors.New("name cannot be empty")
	}
	if c.phone == "" {
		return errors.New("phone cannot be empty")
	}
	return nil
}
