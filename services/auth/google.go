package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"google.golang.org/api/idtoken"
)

var (
	ErrInvalidGoogleToken = errors.New("invalid google token")
)

// VerifyGoogleToken verifies Google Sign-In token and returns sub and email
func VerifyGoogleToken(tokenString string) (sub string, email string, err error) {

	// DEV MODE: Allow demo tokens for testing
	if os.Getenv("APP_ENV") == "dev" && tokenString == "demo-token" {
		return "demo-google-user-123", "demo@google.com", nil
	}

	ctx := context.Background()

	// Validate the token
	payload, err := idtoken.Validate(ctx, tokenString, "")
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidGoogleToken, err)
	}

	// Extract claims
	sub, ok := payload.Claims["sub"].(string)
	if !ok || sub == "" {
		return "", "", errors.New("sub claim not found in token")
	}

	email, ok = payload.Claims["email"].(string)
	if !ok || email == "" {
		return "", "", errors.New("email claim not found in token")
	}

	// Optionally verify email is verified
	emailVerified, ok := payload.Claims["email_verified"].(bool)
	if !ok || !emailVerified {
		return "", "", errors.New("email not verified")
	}

	return sub, email, nil
}
