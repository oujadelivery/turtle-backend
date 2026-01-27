package jwt

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Device string `json:"device"`
	jwt.RegisteredClaims
}

var (
	jwtSecret            = []byte(getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"))
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 60 * 24 * time.Hour // 60 days
)

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// GenerateTokenPair generates both access and refresh tokens
func GenerateTokenPair(userID, role, device string) (*TokenPair, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	if role == "" {
		return nil, errors.New("role is required")
	}

	now := time.Now()
	accessExpiresAt := now.Add(accessTokenDuration)
	refreshExpiresAt := now.Add(refreshTokenDuration)

	// Generate access token
	accessClaims := Claims{
		UserID: userID,
		Role:   role,
		Device: device,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "turtle-delivery",
			Subject:   userID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshClaims := Claims{
		UserID: userID,
		Role:   role,
		Device: device,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "turtle-delivery",
			Subject:   userID,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiresAt,
	}, nil
}

// VerifyToken validates a JWT token and returns claims
func VerifyToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshAccessToken generates a new access token from a valid refresh token
func RefreshAccessToken(refreshTokenString string) (string, error) {
	claims, err := VerifyToken(refreshTokenString)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	// Generate new access token
	tokenPair, err := GenerateTokenPair(claims.UserID, claims.Role, claims.Device)
	if err != nil {
		return "", err
	}

	return tokenPair.AccessToken, nil
}

// ExtractUserID extracts user ID from token without full verification (use carefully)
func ExtractUserID(tokenString string) (string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims.UserID, nil
	}

	return "", errors.New("invalid token claims")
}

// getEnvOrDefault gets environment variable or returns default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
