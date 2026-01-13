package models

import "gorm.io/gorm"

type OtpSession struct {
	gorm.Model
	Target  string // email or phone
	Code    string
	Purpose string // CUSTOMER_LOGIN / CAPTAIN_LOGIN
	Used    bool
}
