package otp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"turtle/infra"
)

const (
	OTPPrefix          = "otp:"
	OTPAttemptPrefix   = "otp:attempt:"
	OTPRateLimitPrefix = "otp:ratelimit:"

	MaxOTPAttempts   = 5
	MaxOTPPerHour    = 3
	OTPAttemptWindow = 15 * time.Minute
	RateLimitWindow  = 1 * time.Hour
)

var (
	ErrOTPExpired         = errors.New("OTP has expired")
	ErrOTPNotFound        = errors.New("OTP not found")
	ErrMaxAttemptsReached = errors.New("maximum OTP attempts reached")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
)

// SaveOTP stores OTP in Redis with expiration
func SaveOTP(target, code string, ttl time.Duration) error {
	ctx := context.Background()
	key := OTPPrefix + target

	err := infra.Redis.Set(ctx, key, code, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to save OTP: %w", err)
	}

	return nil
}

// GetOTP retrieves OTP from Redis
func GetOTP(target string) (string, error) {
	ctx := context.Background()
	key := OTPPrefix + target

	code, err := infra.Redis.Get(ctx, key).Result()
	if err != nil {
		return "", ErrOTPNotFound
	}

	return code, nil
}

// VerifyOTP checks if the provided code matches the stored OTP
func VerifyOTP(target, code string) (bool, error) {
	// Check if max attempts reached
	attempts, err := GetFailedAttempts(target)
	if err == nil && attempts >= MaxOTPAttempts {
		return false, ErrMaxAttemptsReached
	}

	// Get stored OTP
	storedCode, err := GetOTP(target)
	if err != nil {
		return false, err
	}

	// Verify code
	if storedCode != code {
		return false, nil
	}

	return true, nil
}

// DeleteOTP removes OTP from Redis after successful verification
func DeleteOTP(target string) error {
	ctx := context.Background()
	key := OTPPrefix + target
	attemptKey := OTPAttemptPrefix + target

	// Delete both OTP and attempt counter
	pipe := infra.Redis.Pipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, attemptKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete OTP: %w", err)
	}

	return nil
}

// CheckRateLimit checks if user can request another OTP
func CheckRateLimit(target string) (bool, error) {
	ctx := context.Background()
	key := OTPRateLimitPrefix + target

	// Get current count
	count, err := infra.Redis.Get(ctx, key).Int()
	if err != nil && err.Error() != "redis: nil" {
		return false, fmt.Errorf("failed to check rate limit: %w", err)
	}

	// Check if limit exceeded
	if count >= MaxOTPPerHour {
		return true, nil
	}

	return false, nil
}

// IncrementRateLimit increments the rate limit counter
func IncrementRateLimit(target string) error {
	ctx := context.Background()
	key := OTPRateLimitPrefix + target

	// Increment counter
	count, err := infra.Redis.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to increment rate limit: %w", err)
	}

	// Set expiration on first increment
	if count == 1 {
		infra.Redis.Expire(ctx, key, RateLimitWindow)
	}

	return nil
}

// GetFailedAttempts returns the number of failed OTP verification attempts
func GetFailedAttempts(target string) (int, error) {
	ctx := context.Background()
	key := OTPAttemptPrefix + target

	count, err := infra.Redis.Get(ctx, key).Int()
	if err != nil && err.Error() != "redis: nil" {
		return 0, fmt.Errorf("failed to get attempts: %w", err)
	}

	return count, nil
}

// IncrementFailedAttempts increments failed verification attempt counter
func IncrementFailedAttempts(target string) error {
	ctx := context.Background()
	key := OTPAttemptPrefix + target

	// Increment counter
	count, err := infra.Redis.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to increment attempts: %w", err)
	}

	// Set expiration on first increment
	if count == 1 {
		infra.Redis.Expire(ctx, key, OTPAttemptWindow)
	}

	return nil
}

// ResetFailedAttempts clears the failed attempt counter
func ResetFailedAttempts(target string) error {
	ctx := context.Background()
	key := OTPAttemptPrefix + target

	err := infra.Redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to reset attempts: %w", err)
	}

	return nil
}
