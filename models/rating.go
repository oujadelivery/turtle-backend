package models

import (
	"gorm.io/gorm"
)

type Rating struct {
	gorm.Model

	OrderID     uint `gorm:"not null;index"`
	UserID      uint `gorm:"not null;index"` // Who gave the rating
	RatedUserID uint `gorm:"not null;index"` // Who received the rating

	// Rating (1-5 stars)
	Rating float64 `gorm:"type:decimal(2,1);not null"` // 1.0 to 5.0

	// Detailed Ratings (optional)
	BehaviorRating        *float64 `gorm:"type:decimal(2,1)"` // Professionalism
	TimelinessRating      *float64 `gorm:"type:decimal(2,1)"` // On-time delivery
	CommunicationRating   *float64 `gorm:"type:decimal(2,1)"` // Response & updates
	ParcelConditionRating *float64 `gorm:"type:decimal(2,1)"` // How well parcel was handled

	// Feedback
	Comment string `gorm:"type:text"`
	Tags    string `gorm:"type:jsonb"` // Array of predefined tags: ["FRIENDLY", "PROFESSIONAL", "FAST", etc.]

	// Moderation
	IsFlagged  bool   `gorm:"default:false"`
	FlagReason string `gorm:"type:text"`

	// Relationships
	Order        Order `gorm:"foreignKey:OrderID"`
	Reviewer     User  `gorm:"foreignKey:UserID"`
	ReviewedUser User  `gorm:"foreignKey:RatedUserID"`
}

// TableName specifies the table name
func (Rating) TableName() string {
	return "ratings"
}
