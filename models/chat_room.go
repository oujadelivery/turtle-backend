// models/chat_room.go
package models

import (
	"time"

	"gorm.io/gorm"
)

type ChatRoom struct {
	gorm.Model

	OrderID    uint `gorm:"uniqueIndex;not null"`
	CustomerID uint `gorm:"not null;index"`
	CaptainID  uint `gorm:"not null;index"`

	IsActive      bool `gorm:"default:true"`
	LastMessageAt *time.Time

	// Relationships
	Order    Order         `gorm:"foreignKey:OrderID"`
	Customer User          `gorm:"foreignKey:CustomerID"`
	Captain  User          `gorm:"foreignKey:CaptainID"`
	Messages []ChatMessage `gorm:"foreignKey:RoomID"`
}
