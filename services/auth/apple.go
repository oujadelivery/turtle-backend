package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

const applePublicKeyURL = "https://appleid.apple.com/auth/keys"

var (
	ErrInvalidAppleToken = errors.New("invalid apple token")
	ErrAppleKeyFetch     = errors.New("failed to fetch apple public keys")
)

type ApplePublicKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type ApplePublicKeys struct {
	Keys []ApplePublicKey `json:"keys"`
}

type AppleClaims struct {
	Email          string `json:"email"`
	EmailVerified  string `json:"email_verified"`
	IsPrivateEmail string `json:"is_private_email"`
	jwt.RegisteredClaims
}

// VerifyAppleToken verifies Apple Sign-In token and returns sub and email
func VerifyAppleToken(tokenString string) (sub string, email string, err error) {
	// Parse token without verification first to get the header
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &AppleClaims{})
	if err != nil {
		return "", "", fmt.Errorf("failed to parse token: %w", err)
	}
	// DEV MODE: Allow demo tokens for testing
	if os.Getenv("APP_ENV") == "dev" && tokenString == "demo-token" {
		return "demo-apple-user-456", "demo@apple.com", nil
	}

	// Get kid from header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return "", "", errors.New("kid not found in token header")
	}

	// Fetch Apple's public keys
	publicKey, err := getApplePublicKey(kid)
	if err != nil {
		return "", "", err
	}

	// Parse and verify token with the public key
	claims := &AppleClaims{}
	token, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if err != nil || !token.Valid {
		return "", "", ErrInvalidAppleToken
	}

	// Validate claims
	if claims.Subject == "" {
		return "", "", errors.New("sub claim is empty")
	}

	return claims.Subject, claims.Email, nil
}

// getApplePublicKey fetches and parses Apple's public keys
func getApplePublicKey(kid string) (*rsa.PublicKey, error) {
	// Fetch keys from Apple
	resp, err := http.Get(applePublicKeyURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAppleKeyFetch, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var keys ApplePublicKeys
	if err := json.Unmarshal(body, &keys); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keys: %w", err)
	}

	// Find the key with matching kid
	for _, key := range keys.Keys {
		if key.Kid == kid {
			return parseRSAPublicKey(key.N, key.E)
		}
	}

	return nil, errors.New("matching public key not found")
}

// parseRSAPublicKey converts base64url encoded N and E to RSA public key
func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode N: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)

	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}
