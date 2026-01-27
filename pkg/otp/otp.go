package otp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// OTPLength defines the length of OTP codes
const OTPLength = 6

// OTPExpiry defines how long an OTP is valid
const OTPExpiry = 15 * time.Minute

// Purpose defines the purpose of OTP
type Purpose string

const (
	PurposeLogin         Purpose = "LOGIN"
	PurposeVerification  Purpose = "VERIFICATION"
	PurposePasswordReset Purpose = "PASSWORD_RESET"
	PurposeOrderPickup   Purpose = "ORDER_PICKUP"
	PurposeOrderDelivery Purpose = "ORDER_DELIVERY"
)

// OTPSession represents an OTP session
type OTPSession struct {
	Target    string
	Code      string
	Purpose   Purpose
	CreatedAt time.Time
	ExpiresAt time.Time
	Attempts  int
	Used      bool
}

// GenerateOTP generates a new OTP code
func GenerateOTP() (string, error) {
	// Generate cryptographically secure random number
	max := big.NewInt(1000000) // 6 digits: 000000 to 999999
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// Format with leading zeros
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// GenerateAlphanumericOTP generates an alphanumeric OTP (for special cases)
func GenerateAlphanumericOTP(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

// NewOTPSession creates a new OTP session
func NewOTPSession(target string, purpose Purpose) (*OTPSession, error) {
	if target == "" {
		return nil, errors.New("target is required")
	}

	code, err := GenerateOTP()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	return &OTPSession{
		Target:    target,
		Code:      code,
		Purpose:   purpose,
		CreatedAt: now,
		ExpiresAt: now.Add(OTPExpiry),
		Attempts:  0,
		Used:      false,
	}, nil
}

// NewOrderOTPSession creates OTP for order operations (pickup/delivery)
// These are alphanumeric and have longer expiry
func NewOrderOTPSession(orderID string, purpose Purpose) (*OTPSession, error) {
	if orderID == "" {
		return nil, errors.New("order ID is required")
	}

	if purpose != PurposeOrderPickup && purpose != PurposeOrderDelivery {
		return nil, errors.New("invalid purpose for order OTP")
	}

	code, err := GenerateAlphanumericOTP(6)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	return &OTPSession{
		Target:    orderID,
		Code:      code,
		Purpose:   purpose,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour), // 24 hour expiry for order OTPs
		Attempts:  0,
		Used:      false,
	}, nil
}

// Verify verifies an OTP code
func (s *OTPSession) Verify(code string) error {
	// Check if already used
	if s.Used {
		return errors.New("OTP already used")
	}

	// Check if expired
	if time.Now().After(s.ExpiresAt) {
		return errors.New("OTP has expired")
	}

	// Check attempts
	if s.Attempts >= 5 {
		return errors.New("maximum OTP attempts exceeded")
	}

	// Verify code
	if s.Code != code {
		s.Attempts++
		return fmt.Errorf("invalid OTP (attempts: %d/5)", s.Attempts)
	}

	// Mark as used
	s.Used = true
	return nil
}

// IsExpired checks if OTP has expired
func (s *OTPSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// CanRetry checks if user can retry entering OTP
func (s *OTPSession) CanRetry() bool {
	return s.Attempts < 5 && !s.Used && !s.IsExpired()
}

// RemainingTime returns remaining time before expiry
func (s *OTPSession) RemainingTime() time.Duration {
	remaining := time.Until(s.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// FormatOTP formats OTP code for display (e.g., "123-456")
func FormatOTP(code string) string {
	if len(code) == 6 {
		return fmt.Sprintf("%s-%s", code[:3], code[3:])
	}
	return code
}
