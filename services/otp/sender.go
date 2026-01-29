package otp

import "fmt"

func SendEmail(email, code string) error {
	// TODO: Implement with SendGrid / AWS SES
	fmt.Printf("📧 Sending OTP %s to email: %s\n", code, email)
	return nil
}

func SendSMS(phone, code string) error {
	// TODO: Implement with Twilio / Msg91
	fmt.Printf("📱 Sending OTP %s to phone: %s\n", code, phone)
	return nil
}
