package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Provider       string
	ProviderUserID string
	Email          string
	Name           string
	Role           string
}
