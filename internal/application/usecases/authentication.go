package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
	"turtle/pkg/jwt"
	"turtle/pkg/otp"

	"github.com/google/uuid"
)

// AuthenticationService handles all authentication flows
type AuthenticationService struct {
	userRepo         domain.UserRepository
	otpRepo          domain.OTPRepository
	refreshTokenRepo domain.RefreshTokenRepository
	cache            CacheService
}

// CacheService defines caching operations
type CacheService interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error)
}

// NewAuthenticationService creates a new authentication service
func NewAuthenticationService(
	userRepo domain.UserRepository,
	otpRepo domain.OTPRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	cache CacheService,
) *AuthenticationService {
	return &AuthenticationService{
		userRepo:         userRepo,
		otpRepo:          otpRepo,
		refreshTokenRepo: refreshTokenRepo,
		cache:            cache,
	}
}

// ===========================
// CUSTOMER AUTHENTICATION
// ===========================

// SocialLoginInput for Google/Apple login
type SocialLoginInput struct {
	Provider   string // "GOOGLE" or "APPLE"
	ProviderID string // User ID from provider
	Email      string
	FirstName  string
	LastName   string
	ProfilePic string
	DeviceType string // "IOS", "ANDROID", "WEB"
	DeviceInfo map[string]interface{}
}

// SocialLoginOutput contains tokens and user info
type SocialLoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	UserID       string
	IsNewUser    bool
	NeedsPhone   bool // True if user should add phone number
}

// SocialLogin handles Google/Apple login for customers
func (s *AuthenticationService) SocialLogin(ctx context.Context, input SocialLoginInput) (*SocialLoginOutput, error) {
	// Validate input
	if input.Provider != "GOOGLE" && input.Provider != "APPLE" {
		return nil, errors.New("invalid provider")
	}
	if input.ProviderID == "" {
		return nil, errors.New("provider ID is required")
	}
	if input.Email == "" {
		return nil, errors.New("email is required")
	}

	// Check rate limiting (10 login attempts per hour per IP)
	allowed, _, err := s.cache.CheckRateLimit(ctx, "social_login:"+input.Email, 10, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}
	if !allowed {
		return nil, errors.New("too many login attempts, please try again later")
	}

	// Check if user exists by provider ID
	user, err := s.userRepo.FindByProviderID(ctx, input.Provider, input.ProviderID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	isNewUser := false

	if user == nil {
		// Generate UUID for new user
		userID := uuid.New().String()
		
		// New user - create account
		user, err = aggregates.NewUser(
			userID,
			input.FirstName,
			input.LastName,
			&input.Email,
			nil, // Phone is optional for customers
			aggregates.RoleCustomer,
			input.Provider,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// ⭐ For NEW users, use Direct methods that don't increment version
		user.SetProviderID(input.ProviderID)
		
		if input.ProfilePic != "" {
			user.SetProfilePicDirect(input.ProfilePic)
		}
		
		user.VerifyEmailDirect()
		user.UpdateLastActive()  

		// Save to database (version will be 1)
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to save user: %w", err)
		}

		isNewUser = true
	} else {
		// ⭐ Existing user - use normal methods that DO increment version
		user.UpdateProfile(input.FirstName, input.LastName, input.ProfilePic)
		user.UpdateLastActive()
		
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Generate JWT tokens
	tokenPair, err := jwt.GenerateTokenPair(
		user.ID(),
		string(user.PrimaryRole()),
		input.DeviceType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Save refresh token
	if err := s.refreshTokenRepo.Create(
		ctx,
		user.ID(),
		tokenPair.RefreshToken,
		input.DeviceType,
		tokenPair.ExpiresAt.Add(60*24*time.Hour),
	); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &SocialLoginOutput{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		UserID:       user.ID(),
		IsNewUser:    isNewUser,
		NeedsPhone:   user.Phone() == nil,
	}, nil
}

// ===========================
// CAPTAIN AUTHENTICATION (Phone + OTP)
// ===========================

// SendOTPInput for OTP requests
type SendOTPInput struct {
	Phone   string
	Purpose string // "LOGIN" or "VERIFICATION"
}

// SendOTP sends OTP to captain's phone
func (s *AuthenticationService) SendOTP(ctx context.Context, input SendOTPInput) error {
	// Validate phone
	if input.Phone == "" {
		return errors.New("phone number is required")
	}

	// Check rate limiting (3 OTP requests per hour per phone)
	allowed, _, err := s.cache.CheckRateLimit(ctx, "otp_send:"+input.Phone, 3, time.Hour)
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}
	if !allowed {
		return errors.New("too many OTP requests, please try again later")
	}

	// Generate OTP
	otpSession, err := otp.NewOTPSession(input.Phone, otp.Purpose(input.Purpose))
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Save OTP to database
	if err := s.otpRepo.Create(
		ctx,
		otpSession.Target,
		otpSession.Code,
		string(otpSession.Purpose),
		otpSession.ExpiresAt,
	); err != nil {
		return fmt.Errorf("failed to save OTP: %w", err)
	}

	// TODO: Send OTP via SMS (integrate with SMS provider)
	// For now, in development, we can log it
	fmt.Printf("📱 OTP for %s: %s (expires at %s)\n",
		input.Phone,
		otpSession.Code,
		otpSession.ExpiresAt.Format(time.RFC3339),
	)

	return nil
}

// VerifyOTPInput for OTP verification
type VerifyOTPInput struct {
	Phone      string
	Code       string
	Purpose    string // "LOGIN"
	DeviceType string
	DeviceInfo map[string]interface{}
}

// VerifyOTPOutput contains tokens and user info
type VerifyOTPOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	UserID       string
	IsNewUser    bool
	Role         string
}

// VerifyOTPAndLogin verifies OTP and logs in captain (or creates account if new)
func (s *AuthenticationService) VerifyOTPAndLogin(ctx context.Context, input VerifyOTPInput) (*VerifyOTPOutput, error) {
	// Validate input
	if input.Phone == "" {
		return nil, errors.New("phone number is required")
	}
	if input.Code == "" {
		return nil, errors.New("OTP code is required")
	}

	// Check rate limiting (5 verification attempts per 15 minutes)
	allowed, _, err := s.cache.CheckRateLimit(ctx, "otp_verify:"+input.Phone, 5, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}
	if !allowed {
		return nil, errors.New("too many verification attempts, please try again later")
	}

	// Get OTP from database
	otpSession, err := s.otpRepo.FindByTarget(ctx, input.Phone, input.Purpose)
	if err != nil {
		return nil, errors.New("invalid or expired OTP")
	}

	// Verify OTP
	if otpSession.Used {
		return nil, errors.New("OTP already used")
	}
	if time.Now().After(otpSession.ExpiresAt) {
		return nil, errors.New("OTP has expired")
	}
	if otpSession.Attempts >= 5 {
		return nil, errors.New("maximum OTP attempts exceeded")
	}
	if otpSession.Code != input.Code {
		// Increment attempts
		s.otpRepo.IncrementAttempts(ctx, input.Phone, input.Purpose)
		return nil, fmt.Errorf("invalid OTP (attempts: %d/5)", otpSession.Attempts+1)
	}

	// Mark OTP as used
	if err := s.otpRepo.MarkAsUsed(ctx, input.Phone, input.Purpose); err != nil {
		return nil, fmt.Errorf("failed to mark OTP as used: %w", err)
	}

	// Find or create user
	user, err := s.userRepo.FindByPhone(ctx, input.Phone)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	isNewUser := false

	if user == nil {
		// New captain - create account
		user, err = aggregates.NewUser(
			"", // Will be generated
			"",
			"",
			nil, // Email is optional for captains
			&input.Phone,
			aggregates.RoleCaptain, // Default role for phone login
			"PHONE",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Auto-verify phone since OTP was verified
		user.VerifyPhone()

		// Save to database
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to save user: %w", err)
		}

		isNewUser = true
	} else {
		// Existing user - verify phone if not already verified
		if user.Phone() != nil && !user.PhoneVerified() {
			user.VerifyPhone()
		}
	}

	// Update last active
	user.UpdateLastActive()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Generate JWT tokens
	tokenPair, err := jwt.GenerateTokenPair(
		user.ID(),
		string(user.PrimaryRole()),
		input.DeviceType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Save refresh token
	if err := s.refreshTokenRepo.Create(
		ctx,
		user.ID(),
		tokenPair.RefreshToken,
		input.DeviceType,
		tokenPair.ExpiresAt.Add(60*24*time.Hour),
	); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &VerifyOTPOutput{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		UserID:       user.ID(),
		IsNewUser:    isNewUser,
		Role:         string(user.PrimaryRole()),
	}, nil
}

// ===========================
// COMMON OPERATIONS
// ===========================

// AddPhoneNumberInput for adding phone to customer account
type AddPhoneNumberInput struct {
	UserID string
	Phone  string
}

// AddPhoneNumber adds phone number to customer account (who signed up with email)
func (s *AuthenticationService) AddPhoneNumber(ctx context.Context, input AddPhoneNumberInput) error {
	// Get user
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Check if user is customer
	if !user.HasRole(aggregates.RoleCustomer) {
		return errors.New("only customers can add phone numbers this way")
	}

	// Add phone number
	if err := user.AddPhoneNumber(input.Phone); err != nil {
		return fmt.Errorf("failed to add phone number: %w", err)
	}

	// Save to database
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Send verification OTP
	return s.SendOTP(ctx, SendOTPInput{
		Phone:   input.Phone,
		Purpose: "VERIFICATION",
	})
}

// VerifyPhoneInput for verifying added phone number
type VerifyPhoneInput struct {
	UserID string
	Phone  string
	Code   string
}

// VerifyPhone verifies the phone number that was added
func (s *AuthenticationService) VerifyPhone(ctx context.Context, input VerifyPhoneInput) error {
	// Get OTP
	otpSession, err := s.otpRepo.FindByTarget(ctx, input.Phone, "VERIFICATION")
	if err != nil {
		return errors.New("invalid or expired OTP")
	}

	// Verify OTP
	if otpSession.Code != input.Code {
		s.otpRepo.IncrementAttempts(ctx, input.Phone, "VERIFICATION")
		return errors.New("invalid OTP")
	}

	// Mark OTP as used
	if err := s.otpRepo.MarkAsUsed(ctx, input.Phone, "VERIFICATION"); err != nil {
		return err
	}

	// Get user and verify phone
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	user.VerifyPhone()

	return s.userRepo.Update(ctx, user)
}

// ===========================
// DUAL ROLE: CUSTOMER → CAPTAIN
// ===========================

// BecomeCaptainInput for existing customer to become captain
type BecomeCaptainInput struct {
	UserID string
	Phone  string // Required if not already present
}

// BecomeCaptainOutput contains the result
type BecomeCaptainOutput struct {
	Success       bool
	RequiresPhone bool
	RequiresKYC   bool
	Message       string
}

// BecomeCaptain allows an existing customer to add captain role
// This enables dual-role users who can both order and deliver
func (s *AuthenticationService) BecomeCaptain(ctx context.Context, input BecomeCaptainInput) (*BecomeCaptainOutput, error) {
	// Get user
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if already a captain
	if user.HasRole(aggregates.RoleCaptain) {
		kycStatus := aggregates.KYCPending
		if user.CaptainProfile() != nil {
			kycStatus = user.CaptainProfile().KYCStatus()
		}

		return &BecomeCaptainOutput{
			Success:     true,
			RequiresKYC: kycStatus != aggregates.KYCVerified,
			Message:     "User is already a captain",
		}, nil
	}

	// Captain role requires verified phone number
	if user.Phone() == nil || *user.Phone() == "" {
		if input.Phone == "" {
			return &BecomeCaptainOutput{
				Success:       false,
				RequiresPhone: true,
				Message:       "Phone number is required to become a captain",
			}, nil
		}

		// Add phone number
		if err := user.AddPhoneNumber(input.Phone); err != nil {
			return nil, fmt.Errorf("failed to add phone: %w", err)
		}

		// Send verification OTP
		if err := s.SendOTP(ctx, SendOTPInput{
			Phone:   input.Phone,
			Purpose: "VERIFICATION",
		}); err != nil {
			return nil, fmt.Errorf("failed to send OTP: %w", err)
		}

		// Save user
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}

		return &BecomeCaptainOutput{
			Success:       false,
			RequiresPhone: true,
			Message:       "Phone verification OTP sent. Please verify to continue.",
		}, nil
	}

	// Check if phone is verified
	if !user.PhoneVerified() {
		return &BecomeCaptainOutput{
			Success:       false,
			RequiresPhone: true,
			Message:       "Please verify your phone number first",
		}, nil
	}

	// Add captain role
	if err := user.AddRole(aggregates.RoleCaptain); err != nil {
		return nil, fmt.Errorf("failed to add captain role: %w", err)
	}

	// Save user
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &BecomeCaptainOutput{
		Success:     true,
		RequiresKYC: true, // New captains need to complete KYC
		Message:     "Captain role added successfully. Please complete KYC to start delivering.",
	}, nil
}

// SwitchRoleInput for switching active role
type SwitchRoleInput struct {
	UserID     string
	TargetRole string // "CUSTOMER" or "CAPTAIN"
}

// SwitchRole switches the user's active/primary role
// Useful for UI context (showing customer view vs captain view)
func (s *AuthenticationService) SwitchRole(ctx context.Context, input SwitchRoleInput) error {
	// Get user
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	targetRole := aggregates.UserRole(input.TargetRole)

	// Check if user has this role
	if !user.HasRole(targetRole) {
		return errors.New("user does not have this role")
	}

	// For captain role, additional checks
	if targetRole == aggregates.RoleCaptain {
		if user.CaptainProfile() == nil {
			return errors.New("captain profile not initialized")
		}

		// Check KYC status
		if user.CaptainProfile().KYCStatus() != aggregates.KYCVerified {
			return errors.New("captain KYC not verified - cannot switch to captain mode")
		}
	}

	// Note: In your implementation, you might want to track this in session/JWT
	// For now, this is just a validation that the switch is allowed
	// The actual role context would be stored in JWT claims or session

	return nil
}

// RefreshAccessToken refreshes access token using refresh token
func (s *AuthenticationService) RefreshAccessToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	// Verify refresh token
	claims, err := jwt.VerifyToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if token is in database and not revoked
	token, err := s.refreshTokenRepo.FindByToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.New("refresh token not found")
	}
	if token.IsRevoked {
		return nil, errors.New("refresh token has been revoked")
	}

	// Update last used
	s.refreshTokenRepo.UpdateLastUsed(ctx, refreshToken)

	// Generate new access token
	return jwt.GenerateTokenPair(claims.UserID, claims.Role, claims.Device)
}

// Logout revokes refresh token
func (s *AuthenticationService) Logout(ctx context.Context, refreshToken string) error {
	// Revoke refresh token
	if err := s.refreshTokenRepo.Revoke(ctx, refreshToken); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// Optionally blacklist access token in Redis
	// (requires passing access token as well)

	return nil
}
