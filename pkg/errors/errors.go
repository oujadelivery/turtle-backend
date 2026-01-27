package errors

import "fmt"

// ErrorCode represents a specific error code
type ErrorCode string

const (
	// Authentication & Authorization Errors
	CodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	CodeForbidden          ErrorCode = "FORBIDDEN"
	CodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	CodeInvalidToken       ErrorCode = "INVALID_TOKEN"
	CodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	CodeInvalidOTP         ErrorCode = "INVALID_OTP"
	CodeOTPExpired         ErrorCode = "OTP_EXPIRED"
	CodeMaxOTPAttempts     ErrorCode = "MAX_OTP_ATTEMPTS"

	// Validation Errors
	CodeValidationFailed     ErrorCode = "VALIDATION_FAILED"
	CodeInvalidInput         ErrorCode = "INVALID_INPUT"
	CodeMissingRequiredField ErrorCode = "MISSING_REQUIRED_FIELD"
	CodeInvalidFormat        ErrorCode = "INVALID_FORMAT"
	CodeValueOutOfRange      ErrorCode = "VALUE_OUT_OF_RANGE"

	// Resource Errors
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	CodeResourceLocked   ErrorCode = "RESOURCE_LOCKED"
	CodeResourceConflict ErrorCode = "RESOURCE_CONFLICT"

	// Business Logic Errors
	CodeInsufficientBalance ErrorCode = "INSUFFICIENT_BALANCE"
	CodeInvalidState        ErrorCode = "INVALID_STATE"
	CodeOperationNotAllowed ErrorCode = "OPERATION_NOT_ALLOWED"
	CodeKYCNotVerified      ErrorCode = "KYC_NOT_VERIFIED"
	CodeAccountBlocked      ErrorCode = "ACCOUNT_BLOCKED"
	CodeAccountSuspended    ErrorCode = "ACCOUNT_SUSPENDED"

	// Concurrency Errors
	CodeConcurrentModification ErrorCode = "CONCURRENT_MODIFICATION"
	CodeOptimisticLockFailed   ErrorCode = "OPTIMISTIC_LOCK_FAILED"
	CodeDeadlockDetected       ErrorCode = "DEADLOCK_DETECTED"

	// Rate Limiting Errors
	CodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	CodeTooManyRequests   ErrorCode = "TOO_MANY_REQUESTS"

	// System Errors
	CodeInternalError      ErrorCode = "INTERNAL_ERROR"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	CodeTimeout            ErrorCode = "TIMEOUT"
	CodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	CodeCacheError         ErrorCode = "CACHE_ERROR"
)

// AppError is the base error type for application errors
type AppError struct {
	Code       ErrorCode
	Message    string
	Details    map[string]interface{}
	Cause      error
	StatusCode int // HTTP status code
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause adds a cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// ============================================================================
// ERROR CONSTRUCTORS
// ============================================================================

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    make(map[string]interface{}),
	}
}

// ============================================================================
// AUTHENTICATION & AUTHORIZATION ERRORS
// ============================================================================

// ErrUnauthorized indicates unauthorized access
func ErrUnauthorized(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return NewAppError(CodeUnauthorized, message, 401)
}

// ErrForbidden indicates forbidden access
func ErrForbidden(message string) *AppError {
	if message == "" {
		message = "Forbidden - insufficient permissions"
	}
	return NewAppError(CodeForbidden, message, 403)
}

// ErrInvalidCredentials indicates invalid login credentials
func ErrInvalidCredentials() *AppError {
	return NewAppError(
		CodeInvalidCredentials,
		"Invalid credentials",
		401,
	)
}

// ErrInvalidToken indicates an invalid or malformed token
func ErrInvalidToken() *AppError {
	return NewAppError(
		CodeInvalidToken,
		"Invalid or malformed token",
		401,
	)
}

// ErrTokenExpired indicates an expired token
func ErrTokenExpired() *AppError {
	return NewAppError(
		CodeTokenExpired,
		"Token has expired",
		401,
	)
}

// ErrInvalidOTP indicates an invalid OTP
func ErrInvalidOTP(attempts int, maxAttempts int) *AppError {
	return NewAppError(
		CodeInvalidOTP,
		fmt.Sprintf("Invalid OTP (attempts: %d/%d)", attempts, maxAttempts),
		400,
	).WithDetail("attempts", attempts).
		WithDetail("maxAttempts", maxAttempts)
}

// ErrOTPExpired indicates an expired OTP
func ErrOTPExpired() *AppError {
	return NewAppError(
		CodeOTPExpired,
		"OTP has expired",
		400,
	)
}

// ErrMaxOTPAttempts indicates maximum OTP attempts exceeded
func ErrMaxOTPAttempts() *AppError {
	return NewAppError(
		CodeMaxOTPAttempts,
		"Maximum OTP verification attempts exceeded",
		429,
	)
}

// ============================================================================
// VALIDATION ERRORS
// ============================================================================

// ErrValidation creates a validation error
func ErrValidation(message string) *AppError {
	return NewAppError(CodeValidationFailed, message, 400)
}

// ErrInvalidInput creates an invalid input error
func ErrInvalidInput(field, message string) *AppError {
	return NewAppError(
		CodeInvalidInput,
		fmt.Sprintf("Invalid input for field '%s': %s", field, message),
		400,
	).WithDetail("field", field)
}

// ErrMissingRequiredField creates a missing field error
func ErrMissingRequiredField(field string) *AppError {
	return NewAppError(
		CodeMissingRequiredField,
		fmt.Sprintf("Required field '%s' is missing", field),
		400,
	).WithDetail("field", field)
}

// ErrInvalidFormat creates an invalid format error
func ErrInvalidFormat(field, expectedFormat string) *AppError {
	return NewAppError(
		CodeInvalidFormat,
		fmt.Sprintf("Invalid format for field '%s': expected %s", field, expectedFormat),
		400,
	).WithDetail("field", field).
		WithDetail("expectedFormat", expectedFormat)
}

// ErrValueOutOfRange creates an out of range error
func ErrValueOutOfRange(field string, min, max, actual interface{}) *AppError {
	return NewAppError(
		CodeValueOutOfRange,
		fmt.Sprintf("Value for field '%s' is out of range", field),
		400,
	).WithDetail("field", field).
		WithDetail("min", min).
		WithDetail("max", max).
		WithDetail("actual", actual)
}

// ============================================================================
// RESOURCE ERRORS
// ============================================================================

// ErrNotFound creates a not found error
func ErrNotFound(resource string) *AppError {
	return NewAppError(
		CodeNotFound,
		fmt.Sprintf("%s not found", resource),
		404,
	).WithDetail("resource", resource)
}

// ErrAlreadyExists creates an already exists error
func ErrAlreadyExists(resource, identifier string) *AppError {
	return NewAppError(
		CodeAlreadyExists,
		fmt.Sprintf("%s already exists: %s", resource, identifier),
		409,
	).WithDetail("resource", resource).
		WithDetail("identifier", identifier)
}

// ErrResourceLocked creates a resource locked error
func ErrResourceLocked(resource string) *AppError {
	return NewAppError(
		CodeResourceLocked,
		fmt.Sprintf("%s is currently locked", resource),
		423,
	).WithDetail("resource", resource)
}

// ErrResourceConflict creates a resource conflict error
func ErrResourceConflict(message string) *AppError {
	return NewAppError(CodeResourceConflict, message, 409)
}

// ============================================================================
// BUSINESS LOGIC ERRORS
// ============================================================================

// ErrInsufficientBalance creates an insufficient balance error
func ErrInsufficientBalance(required, available int64) *AppError {
	return NewAppError(
		CodeInsufficientBalance,
		"Insufficient wallet balance",
		400,
	).WithDetail("required", required).
		WithDetail("available", available)
}

// ErrInvalidState creates an invalid state error
func ErrInvalidState(currentState, operation string) *AppError {
	return NewAppError(
		CodeInvalidState,
		fmt.Sprintf("Cannot perform '%s' in current state: %s", operation, currentState),
		400,
	).WithDetail("currentState", currentState).
		WithDetail("operation", operation)
}

// ErrOperationNotAllowed creates an operation not allowed error
func ErrOperationNotAllowed(operation, reason string) *AppError {
	return NewAppError(
		CodeOperationNotAllowed,
		fmt.Sprintf("Operation '%s' not allowed: %s", operation, reason),
		403,
	).WithDetail("operation", operation).
		WithDetail("reason", reason)
}

// ErrKYCNotVerified creates a KYC not verified error
func ErrKYCNotVerified() *AppError {
	return NewAppError(
		CodeKYCNotVerified,
		"KYC verification is required to perform this action",
		403,
	)
}

// ErrAccountBlocked creates an account blocked error
func ErrAccountBlocked(reason string) *AppError {
	return NewAppError(
		CodeAccountBlocked,
		"Account has been blocked",
		403,
	).WithDetail("reason", reason)
}

// ErrAccountSuspended creates an account suspended error
func ErrAccountSuspended(reason string) *AppError {
	return NewAppError(
		CodeAccountSuspended,
		"Account has been suspended",
		403,
	).WithDetail("reason", reason)
}

// ============================================================================
// CONCURRENCY ERRORS
// ============================================================================

// ErrConcurrentModification creates a concurrent modification error
func ErrConcurrentModification() *AppError {
	return NewAppError(
		CodeConcurrentModification,
		"Resource was modified by another request. Please retry.",
		409,
	)
}

// ErrOptimisticLockFailed creates an optimistic lock failure error
func ErrOptimisticLockFailed(resource string, expectedVersion, actualVersion int) *AppError {
	return NewAppError(
		CodeOptimisticLockFailed,
		fmt.Sprintf("Optimistic lock failed for %s", resource),
		409,
	).WithDetail("resource", resource).
		WithDetail("expectedVersion", expectedVersion).
		WithDetail("actualVersion", actualVersion)
}

// ErrDeadlockDetected creates a deadlock error
func ErrDeadlockDetected() *AppError {
	return NewAppError(
		CodeDeadlockDetected,
		"Deadlock detected. Please retry the operation.",
		409,
	)
}

// ============================================================================
// RATE LIMITING ERRORS
// ============================================================================

// ErrRateLimitExceeded creates a rate limit exceeded error
func ErrRateLimitExceeded(operation string, retryAfter int) *AppError {
	return NewAppError(
		CodeRateLimitExceeded,
		fmt.Sprintf("Rate limit exceeded for %s", operation),
		429,
	).WithDetail("operation", operation).
		WithDetail("retryAfter", retryAfter)
}

// ErrTooManyRequests creates a too many requests error
func ErrTooManyRequests(message string) *AppError {
	if message == "" {
		message = "Too many requests. Please try again later."
	}
	return NewAppError(CodeTooManyRequests, message, 429)
}

// ============================================================================
// SYSTEM ERRORS
// ============================================================================

// ErrInternal creates an internal error (hides details from user)
func ErrInternal(cause error) *AppError {
	return NewAppError(
		CodeInternalError,
		"An internal error occurred. Please try again later.",
		500,
	).WithCause(cause)
}

// ErrServiceUnavailable creates a service unavailable error
func ErrServiceUnavailable(service string) *AppError {
	return NewAppError(
		CodeServiceUnavailable,
		fmt.Sprintf("Service temporarily unavailable: %s", service),
		503,
	).WithDetail("service", service)
}

// ErrTimeout creates a timeout error
func ErrTimeout(operation string) *AppError {
	return NewAppError(
		CodeTimeout,
		fmt.Sprintf("Operation timed out: %s", operation),
		504,
	).WithDetail("operation", operation)
}

// ErrDatabase creates a database error
func ErrDatabase(cause error) *AppError {
	return NewAppError(
		CodeDatabaseError,
		"Database operation failed",
		500,
	).WithCause(cause)
}

// ErrCache creates a cache error
func ErrCache(cause error) *AppError {
	return NewAppError(
		CodeCacheError,
		"Cache operation failed",
		500,
	).WithCause(cause)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts AppError from error
func GetAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}

// GetErrorCode extracts error code from error
func GetErrorCode(err error) ErrorCode {
	if appErr, ok := GetAppError(err); ok {
		return appErr.Code
	}
	return CodeInternalError
}

// GetStatusCode extracts HTTP status code from error
func GetStatusCode(err error) int {
	if appErr, ok := GetAppError(err); ok {
		return appErr.StatusCode
	}
	return 500
}
