package graph

import (
	"fmt"
	"time"

	"turtle/db"
	"turtle/graph/model"
	"turtle/models"
	"turtle/pkg/jwt"
	"turtle/services/address"
	"turtle/services/chat"
	"turtle/services/location"
	"turtle/services/notification"
	"turtle/services/order"
	"turtle/services/payment"
	"turtle/services/pricing"
	"turtle/services/rating"
	"turtle/services/realtime"
	"turtle/services/support"
	"turtle/services/user"
)

// This file will not be regenerated automatically.
// It serves as dependency injection for your app, add any dependencies you require here.


type Resolver struct {
    UserService         *user.Service
	AddressService      *address.Service
	OrderService        *order.Service
	PaymentService      *payment.Service
	PricingService      *pricing.Service
	RatingService       *rating.Service
	LocationService     *location.Service
	NotificationService *notification.Service
	SupportService      *support.Service
	ChatService         *chat.Service
	RealtimeService     *realtime.PubSubService
}

func NewResolver() *Resolver {
    return &Resolver{
        UserService:         user.NewService(),
		AddressService:      address.NewService(),
		OrderService:        order.NewService(),
		PaymentService:      payment.NewService(),
		PricingService:      pricing.NewService(),
		RatingService:       rating.NewService(),
		LocationService:     location.NewService(),
		NotificationService: notification.NewService(),
		SupportService:      support.NewService(),
		ChatService:         chat.NewService(),
		RealtimeService:     realtime.NewPubSubService(),
    }
}

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