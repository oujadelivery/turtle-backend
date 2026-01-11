package auth

import (
	"turtle/db"
	"turtle/models"
)

func SocialLogin(provider, providerID string) *models.User {
	var user models.User
	db.DB.FirstOrCreate(&user, models.User{ProviderUserID: providerID})
	return &user
}
