package auth

import (
	"errors"
	"fmt"

	"turtle/db"
	"turtle/models"
)

var (
    ErrUserCreationFailed = errors.New("failed to create user")
    ErrInvalidEmail       = errors.New("invalid email address")
)

// SocialLogin handles user creation/retrieval for social login
func SocialLogin(provider, sub, email string) (*models.User, error) {
    
    // Validate email
    if email == "" {
        return nil, ErrInvalidEmail
    }
    
    var user models.User
    
    // Build the provider ID (e.g., "GOOGLE:123456789")
    providerID := fmt.Sprintf("%s:%s", provider, sub)
    
    // Try to find existing user by provider ID
    err := db.DB.Where("provider_id = ?", providerID).First(&user).Error
    
    if err != nil {
        // User doesn't exist with this provider ID, try to find by email
        err = db.DB.Where("email = ?", email).First(&user).Error
        
        if err != nil {
            // Create new user
            user = models.User{
                Email:      email,
                ProviderID: providerID,
                Provider:   provider,
                Role:       "CUSTOMER",
                Status:     "ACTIVE",
                EmailVerified: true, // Social login emails are pre-verified
            }
            
            if err := db.DB.Create(&user).Error; err != nil {
                return nil, fmt.Errorf("%w: %v", ErrUserCreationFailed, err)
            }
            
            return &user, nil
        }
        
        // User exists with email but different provider, link the account
        if user.ProviderID == "" {
            user.ProviderID = providerID
            user.Provider = provider
            user.EmailVerified = true
            
            if err := db.DB.Save(&user).Error; err != nil {
                return nil, fmt.Errorf("failed to link social account: %w", err)
            }
        }
    }
    
    return &user, nil
}

// GetUserByID retrieves user by ID
func GetUserByID(userID uint) (*models.User, error) {
    var user models.User
    
    if err := db.DB.First(&user, userID).Error; err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    
    return &user, nil
}

// GetUserByEmail retrieves user by email
func GetUserByEmail(email string) (*models.User, error) {
    var user models.User
    
    if err := db.DB.Where("email = ?", email).First(&user).Error; err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    
    return &user, nil
}

// GetUserByPhone retrieves user by phone
func GetUserByPhone(phone string) (*models.User, error) {
    var user models.User
    
    if err := db.DB.Where("phone = ?", phone).First(&user).Error; err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    
    return &user, nil
}