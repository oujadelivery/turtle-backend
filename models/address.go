package models

import (
	"time"

	"gorm.io/gorm"
)

type Address struct {
	gorm.Model

	UserID uint `gorm:"not null;index"` // Foreign key to User

	// Address Details
	Label        string `gorm:"size:50"` // HOME / WORK / OTHER
	AddressLine1 string `gorm:"size:255;not null"`
	AddressLine2 string `gorm:"size:255"`
	Landmark     string `gorm:"size:255"`
	City         string `gorm:"size:100;not null;index"`
	State        string `gorm:"size:100;not null;index"`
	Country      string `gorm:"size:100;not null;default:'India'"`
	PostalCode   string `gorm:"size:20;index"`

	// Geolocation
	Latitude  float64 `gorm:"type:decimal(10,8);not null"`
	Longitude float64 `gorm:"type:decimal(11,8);not null"`

	// Contact at this address
	ContactName  string `gorm:"size:100"`
	ContactPhone string `gorm:"size:20"`

	// Usage Analytics
	IsDefault        bool       `gorm:"default:false;index"` // Default address for user
	UsageCount       int        `gorm:"default:0"`           // How many times this address was used
	LastUsedAt       *time.Time `gorm:"index"`               // When was this address last used
	LastUsedForOrder uint       // Order ID that last used this address

	// Time-based Analytics (for smart suggestions)
	MorningUsageCount   int `gorm:"default:0"` // 6 AM - 12 PM
	AfternoonUsageCount int `gorm:"default:0"` // 12 PM - 6 PM
	EveningUsageCount   int `gorm:"default:0"` // 6 PM - 12 AM
	NightUsageCount     int `gorm:"default:0"` // 12 AM - 6 AM

	// Day-based Analytics
	WeekdayUsageCount int `gorm:"default:0"` // Mon-Fri
	WeekendUsageCount int `gorm:"default:0"` // Sat-Sun

	// Status
	IsActive   bool `gorm:"default:true;index"`
	IsVerified bool `gorm:"default:false"` // Whether location coordinates are verified

	// Relationships
	User           User    `gorm:"foreignKey:UserID"`
	PickupOrders   []Order `gorm:"foreignKey:PickupAddressID"`
	DeliveryOrders []Order `gorm:"foreignKey:DeliveryAddressID"`
}

// TableName specifies the table name
func (Address) TableName() string {
	return "addresses"
}

// GetFullAddress returns formatted full address
func (a *Address) GetFullAddress() string {
	address := a.AddressLine1
	if a.AddressLine2 != "" {
		address += ", " + a.AddressLine2
	}
	if a.Landmark != "" {
		address += ", " + a.Landmark
	}
	address += ", " + a.City + ", " + a.State + " - " + a.PostalCode
	return address
}

// IncrementUsage updates usage statistics based on current time
func (a *Address) IncrementUsage() {
	now := time.Now()
	hour := now.Hour()

	// Increment total usage
	a.UsageCount++
	a.LastUsedAt = &now

	// Time-based increment
	switch {
	case hour >= 6 && hour < 12:
		a.MorningUsageCount++
	case hour >= 12 && hour < 18:
		a.AfternoonUsageCount++
	case hour >= 18 && hour < 24:
		a.EveningUsageCount++
	default:
		a.NightUsageCount++
	}

	// Day-based increment
	weekday := now.Weekday()
	if weekday >= time.Monday && weekday <= time.Friday {
		a.WeekdayUsageCount++
	} else {
		a.WeekendUsageCount++
	}
}

// GetPreferredTimeSlot returns when this address is most frequently used
func (a *Address) GetPreferredTimeSlot() string {
	counts := map[string]int{
		"MORNING":   a.MorningUsageCount,
		"AFTERNOON": a.AfternoonUsageCount,
		"EVENING":   a.EveningUsageCount,
		"NIGHT":     a.NightUsageCount,
	}

	max := 0
	preferred := "MORNING"
	for slot, count := range counts {
		if count > max {
			max = count
			preferred = slot
		}
	}
	return preferred
}
