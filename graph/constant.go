package graph

import (
	"errors"
	"time"
)

// Device types
const (
	DeviceMobile  = "MOBILE"
	DeviceWeb     = "WEB"
	DeviceDesktop = "DESKTOP"
)

// Token TTL
const (
	RefreshTokenTTL = 60 * 24 * time.Hour // 60 days
	AccessTokenTTL  = 15 * time.Minute    // 15 minutes
)

// User roles
const (
	RoleCustomer = "CUSTOMER"
	RoleCaptain  = "CAPTAIN"
	RoleAdmin    = "ADMIN"
)

// OTP purposes
const (
	PurposeCustomerLogin = "CUSTOMER_LOGIN"
	PurposeCaptainLogin  = "CAPTAIN_LOGIN"
)

// Common errors
var (
	ErrInvalidProvider    = errors.New("unsupported provider")
	ErrInvalidOTP         = errors.New("invalid or expired OTP")
	ErrInvalidRefresh     = errors.New("invalid or expired refresh token")
	ErrInvalidPurpose     = errors.New("invalid purpose")
	ErrInvalidDevice      = errors.New("invalid device type")
	ErrOTPNotFound        = errors.New("OTP not found or expired")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded, please try again later")
	ErrMaxAttemptsReached = errors.New("maximum OTP verification attempts reached")
	ErrUserNotFound       = errors.New("user not found")
)

// IsValidDevice checks if device string is valid
func IsValidDevice(device string) bool {
	switch device {
	case DeviceMobile, DeviceWeb, DeviceDesktop:
		return true
	default:
		return false
	}
}

// IsValidRole checks if role string is valid
func IsValidRole(role string) bool {
	switch role {
	case RoleCustomer, RoleCaptain, RoleAdmin:
		return true
	default:
		return false
	}
}

// IsValidPurpose checks if purpose string is valid
func IsValidPurpose(purpose string) bool {
	switch purpose {
	case PurposeCustomerLogin, PurposeCaptainLogin:
		return true
	default:
		return false
	}
}
