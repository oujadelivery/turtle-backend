package domain

import "errors"

// Common domain errors
var (
	// ErrNotFound indicates that a requested entity was not found
	ErrNotFound = errors.New("entity not found")

	// ErrAlreadyExists indicates that an entity already exists
	ErrAlreadyExists = errors.New("entity already exists")

	// ErrInvalidInput indicates invalid input data
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates unauthorized access
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates forbidden access
	ErrForbidden = errors.New("forbidden")

	// ErrConcurrentModification indicates optimistic locking conflict
	ErrConcurrentModification = errors.New("concurrent modification detected")

	// ErrInsufficientBalance indicates insufficient wallet balance
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrInvalidState indicates invalid state transition
	ErrInvalidState = errors.New("invalid state")
)
