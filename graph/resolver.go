package graph

import (
	"fmt"
	"time"

	"turtle/db"
	"turtle/graph/model"
	"turtle/models"
	"turtle/pkg/jwt"
)

// This file will not be regenerated automatically.
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}

// createAuthSession creates a new authentication session for a user
func (r *Resolver) createAuthSession(user *models.User, device string) (*model.AuthPayload, error) {

    // Validate device
    if !IsValidDevice(device) {
        return nil, ErrInvalidDevice
    }

    // Generate token pair
    access, refresh := jwt.GeneratePair(user.ID, user.Role, device)

    // Create refresh token session
    session := &models.RefreshToken{
        UserID:    user.ID,
        Token:     refresh,
        Device:    device,
        ExpiresAt: time.Now().Add(RefreshTokenTTL),
    }

    if err := db.DB.Create(session).Error; err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }

    return &model.AuthPayload{
        AccessToken:  access,
        RefreshToken: refresh,
        UserID:       int(user.ID),
        Role:         user.Role,
    }, nil
}