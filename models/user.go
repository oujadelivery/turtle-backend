package models

import "gorm.io/gorm"

type User struct {
    gorm.Model

    // Common
    Role string `gorm:"not null;default:'CUSTOMER'"` // CUSTOMER / CAPTAIN / ADMIN

    // Customer fields
    Email         string `gorm:"uniqueIndex"`
    EmailVerified bool

    // Captain fields
    Phone         string `gorm:"uniqueIndex"`
    PhoneVerified bool

    // Social login
    Provider   string
    ProviderID string `gorm:"uniqueIndex"` // Format: "GOOGLE:123456789"

    Status string `gorm:"default:'ACTIVE'"` // ACTIVE / BLOCKED / PENDING_KYC
}