package models

import (
	"time"

	"gorm.io/gorm"
)

type RefreshToken struct {
	gorm.Model
	UserID    uint
	Token     string `gorm:"uniqueIndex"`
	Device    string
	ExpiresAt time.Time
}
