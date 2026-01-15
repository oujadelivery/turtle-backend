package models

import (
	"time"

	"gorm.io/gorm"
)

// LocationTracking stores real-time location history for captains during delivery
type LocationTracking struct {
	gorm.Model

	OrderID   uint `gorm:"not null;index"`
	CaptainID uint `gorm:"not null;index"`

	// Location
	Latitude  float64 `gorm:"type:decimal(10,8);not null"`
	Longitude float64 `gorm:"type:decimal(11,8);not null"`
	Accuracy  float64 `gorm:"type:decimal(8,2)"` // in meters
	Altitude  float64 `gorm:"type:decimal(8,2)"` // in meters

	// Movement
	Speed   float64 `gorm:"type:decimal(6,2)"` // in km/h
	Bearing float64 `gorm:"type:decimal(6,2)"` // direction in degrees (0-360)

	// Timestamp
	RecordedAt time.Time `gorm:"not null;index"`

	// Battery & Network (for reliability)
	BatteryLevel int    `gorm:"type:smallint"` // 0-100
	NetworkType  string `gorm:"size:20"`       // 4G / 5G / WIFI
	IsGPSEnabled bool   `gorm:"default:true"`

	// Relationships
	Order   Order `gorm:"foreignKey:OrderID"`
	Captain User  `gorm:"foreignKey:CaptainID"`
}

// TableName specifies the table name
func (LocationTracking) TableName() string {
	return "location_tracking"
}

// CurrentLocation stores user's current/last known location (separate from real-time tracking)
type CurrentLocation struct {
	gorm.Model

	UserID uint `gorm:"uniqueIndex;not null"` // One location per user

	// Location
	Latitude  float64 `gorm:"type:decimal(10,8);not null"`
	Longitude float64 `gorm:"type:decimal(11,8);not null"`
	Accuracy  float64 `gorm:"type:decimal(8,2)"` // in meters

	// Address (geocoded)
	FormattedAddress string `gorm:"type:text"`
	City             string `gorm:"size:100;index"`
	State            string `gorm:"size:100;index"`
	Country          string `gorm:"size:100"`
	PostalCode       string `gorm:"size:20"`

	// How location was set
	Source string `gorm:"size:20"` // GPS / MANUAL / ADDRESS

	// Is user actively sharing location
	IsLiveTracking bool `gorm:"default:false"`

	// Timestamps
	UpdatedAt time.Time `gorm:"not null;index"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name
func (CurrentLocation) TableName() string {
	return "current_locations"
}
