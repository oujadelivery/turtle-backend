package models

import (
	"time"

	"gorm.io/gorm"
)

type ChatMessage struct {
	gorm.Model

	RoomID     uint `gorm:"not null;index"`
	SenderID   uint `gorm:"not null;index"`
	ReceiverID uint `gorm:"not null;index"`

	Message     string `gorm:"type:text;not null"`
	MessageType string `gorm:"size:20;default:'TEXT'"` // TEXT / IMAGE / LOCATION / AUDIO

	IsRead      bool `gorm:"default:false"`
	ReadAt      *time.Time
	DeliveredAt *time.Time

	// For media messages
	MediaURL     string `gorm:"size:500"`
	ThumbnailURL string `gorm:"size:500"`

	// For location sharing
	Latitude  float64 `gorm:"type:decimal(10,8)"`
	Longitude float64 `gorm:"type:decimal(11,8)"`

	// Relationships
	Room     ChatRoom `gorm:"foreignKey:RoomID"`
	Sender   User     `gorm:"foreignKey:SenderID"`
	Receiver User     `gorm:"foreignKey:ReceiverID"`
}
