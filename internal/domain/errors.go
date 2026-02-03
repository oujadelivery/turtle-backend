package domain

import "errors"

// ============================================================================
// GENERIC DOMAIN ERRORS
// ============================================================================

var (
	// Resource errors
	ErrNotFound               = errors.New("resource not found")
	ErrAlreadyExists          = errors.New("resource already exists")
	ErrConcurrentModification = errors.New("concurrent modification detected")
	ErrInvalidState           = errors.New("invalid state for operation")
	ErrDeleted                = errors.New("resource has been deleted")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("forbidden")
	
	// Validation errors
	ErrInvalidInput           = errors.New("invalid input")
	ErrRequiredField          = errors.New("required field missing")
	ErrInvalidFormat          = errors.New("invalid format")
	ErrOutOfRange             = errors.New("value out of range")
)

// ============================================================================
// ORDER-SPECIFIC ERRORS
// ============================================================================

var (
	// State machine errors
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrOrderAlreadyCompleted  = errors.New("order already completed")
	ErrOrderAlreadyCancelled  = errors.New("order already cancelled")
	ErrOrderInTerminalState   = errors.New("order is in terminal state")
	
	// Assignment errors
	ErrSelfAssignment         = errors.New("user cannot be both booker and captain")
	ErrCaptainNotAssigned     = errors.New("captain not assigned to order")
	ErrCaptainAlreadyAssigned = errors.New("captain already assigned to order")
	ErrWrongCaptain           = errors.New("only assigned captain can perform this action")
	
	// OTP errors
	ErrInvalidOTP             = errors.New("invalid OTP")
	ErrOTPExpired             = errors.New("OTP has expired")
	ErrOTPNotGenerated        = errors.New("OTP not generated yet")
	
	// Cancellation errors
	ErrCannotCancelAfterPickup = errors.New("cannot cancel order after pickup")
	ErrUnauthorizedCancellation = errors.New("user not authorized to cancel this order")
	ErrInvalidCancellationState = errors.New("order cannot be cancelled in current state")
	
	// Pricing errors
	ErrPricingNotSet           = errors.New("pricing not set for order")
	ErrInvalidPricing          = errors.New("invalid pricing configuration")
	ErrNegativeAmount          = errors.New("amount cannot be negative")
	
	// Waiting time errors
	ErrWaitingNotStarted       = errors.New("waiting timer not started")
	ErrWaitingAlreadyStarted   = errors.New("waiting timer already started")
)

// ============================================================================
// CAPTAIN-SPECIFIC ERRORS
// ============================================================================

var (
	ErrCaptainNotAvailable    = errors.New("captain not available")
	ErrCaptainNotVerified     = errors.New("captain KYC not verified")
	ErrCaptainHasActiveOrder  = errors.New("captain already has active order")
	ErrCaptainSuspended       = errors.New("captain account is suspended")
	ErrCaptainNotOnline       = errors.New("captain is not online")
	ErrCaptainTooFar          = errors.New("captain is too far from pickup location")
)

// ============================================================================
// PAYMENT ERRORS
// ============================================================================

var (
	ErrInsufficientFunds      = errors.New("insufficient wallet balance")
	ErrPaymentFailed          = errors.New("payment processing failed")
	ErrPaymentAlreadyCaptured = errors.New("payment already captured")
	ErrRefundFailed           = errors.New("refund processing failed")
	ErrInvalidPaymentMethod   = errors.New("invalid payment method")
	ErrPaymentIntentNotFound  = errors.New("payment intent not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// ============================================================================
// USER ERRORS
// ============================================================================

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrUserSuspended          = errors.New("user account is suspended")
	ErrUserBlocked            = errors.New("user account is blocked")
	ErrUserNotCustomer        = errors.New("user is not a customer")
	ErrUserNotCaptain         = errors.New("user is not a captain")
	ErrUserNotAdmin           = errors.New("user is not an admin")
)

// ============================================================================
// PENALTY & REWARD ERRORS
// ============================================================================

var (
	ErrNegativePenaltyPoints  = errors.New("penalty points cannot be negative")
	ErrNegativeRewardPoints   = errors.New("reward points cannot be negative")
	ErrPenaltyRecordNotFound  = errors.New("penalty record not found")
)

// ============================================================================
// LOCATION ERRORS
// ============================================================================

var (
	ErrInvalidLocation        = errors.New("invalid location coordinates")
	ErrLocationNotFound       = errors.New("location not found")
	ErrOutsideServiceArea     = errors.New("location outside service area")
)

// ============================================================================
// TIME-RELATED ERRORS
// ============================================================================

var (
	ErrScheduledTimeInPast    = errors.New("scheduled time is in the past")
	ErrScheduledTimeTooFar    = errors.New("scheduled time is too far in the future")
	ErrTimeoutExpired         = errors.New("timeout expired")
)

// ============================================================================
// PROMO CODE ERRORS
// ============================================================================

var (
	ErrPromoCodeNotFound      = errors.New("promo code not found")
	ErrPromoCodeExpired       = errors.New("promo code has expired")
	ErrPromoCodeNotActive     = errors.New("promo code is not active")
	ErrPromoCodeUsageLimitReached = errors.New("promo code usage limit reached")
	ErrPromoCodeMinOrderNotMet = errors.New("order amount does not meet promo code minimum")
)

// ============================================================================
// RATE LIMITING ERRORS
// ============================================================================

var (
	ErrRateLimitExceeded      = errors.New("rate limit exceeded")
	ErrTooManyAttempts        = errors.New("too many attempts")
)

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// IsNotFoundError checks if error is a not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrCaptainNotAssigned) ||
		errors.Is(err, ErrPaymentIntentNotFound) ||
		errors.Is(err, ErrPromoCodeNotFound) ||
		errors.Is(err, ErrPenaltyRecordNotFound) ||
		errors.Is(err, ErrLocationNotFound)
}

// IsConcurrencyError checks if error is a concurrency error
func IsConcurrencyError(err error) bool {
	return errors.Is(err, ErrConcurrentModification)
}

// IsValidationError checks if error is a validation error
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, ErrRequiredField) ||
		errors.Is(err, ErrInvalidFormat) ||
		errors.Is(err, ErrOutOfRange) ||
		errors.Is(err, ErrInvalidLocation) ||
		errors.Is(err, ErrNegativeAmount)
}

// IsAuthorizationError checks if error is an authorization error
func IsAuthorizationError(err error) bool {
	return errors.Is(err, ErrUnauthorizedCancellation) ||
		errors.Is(err, ErrWrongCaptain) ||
		errors.Is(err, ErrUserNotAdmin) ||
		errors.Is(err, ErrUserNotCaptain) ||
		errors.Is(err, ErrUserNotCustomer)
}

// IsBusinessRuleError checks if error is a business rule violation
func IsBusinessRuleError(err error) bool {
	return errors.Is(err, ErrSelfAssignment) ||
		errors.Is(err, ErrInvalidStateTransition) ||
		errors.Is(err, ErrCannotCancelAfterPickup) ||
		errors.Is(err, ErrCaptainHasActiveOrder) ||
		errors.Is(err, ErrPromoCodeMinOrderNotMet)
}