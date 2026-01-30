package aggregates

import (
	"errors"
	"time"

	"turtle/internal/domain/events"
	"turtle/internal/domain/valueobjects"
)

// UserRole represents the role of a user
type UserRole string

const (
	RoleCustomer UserRole = "CUSTOMER"
	RoleCaptain  UserRole = "CAPTAIN"
	RoleAdmin    UserRole = "ADMIN"
)

// UserStatus represents user account status
type UserStatus string

const (
	StatusActive      UserStatus = "ACTIVE"
	StatusBlocked     UserStatus = "BLOCKED"
	StatusPendingKYC  UserStatus = "PENDING_KYC"
	StatusSuspended   UserStatus = "SUSPENDED"
	StatusDeactivated UserStatus = "DEACTIVATED"
)

// KYCStatus represents KYC verification status (for captains)
type KYCStatus string

const (
	KYCPending  KYCStatus = "PENDING"
	KYCVerified KYCStatus = "VERIFIED"
	KYCRejected KYCStatus = "REJECTED"
)

// VehicleType represents type of vehicle (for captains)
type VehicleType string

const (
	VehicleBike  VehicleType = "BIKE"
	VehicleCar   VehicleType = "CAR"
	VehicleVan   VehicleType = "VAN"
	VehicleTruck VehicleType = "TRUCK"
)

// User aggregate root
type User struct {
	// Identity
	id      string
	version int

	// Profile
	firstName  string
	lastName   string
	profilePic string

	// Contact
	email         *string
	emailVerified bool
	phone         *string
	phoneVerified bool

	// Role & Status
	primaryRole UserRole
	roles       []UserRole // A user can have multiple roles
	status      UserStatus

	// Authentication
	provider   string // GOOGLE, APPLE, PHONE, EMAIL
	providerID *string

	// Captain-specific fields (only populated if user has CAPTAIN role)
	captainProfile *CaptainProfile

	// Admin-specific fields (only populated if user has ADMIN role)
	adminProfile *AdminProfile

	// Wallet
	walletBalance  *valueobjects.Money
	walletVersion  int // For optimistic locking on wallet operations
	walletCurrency string

	// Stats (applicable to all roles)
	totalOrders     int
	totalDeliveries int // For captains
	rating          float64
	totalRatings    int

	// Metadata
	lastActiveAt *time.Time
	createdAt    time.Time
	updatedAt    time.Time
	deletedAt    *time.Time

	// Domain events (not persisted)
	pendingEvents []events.DomainEvent
}

// CaptainProfile contains captain-specific data
type CaptainProfile struct {
	// Vehicle
	vehicleType   VehicleType
	vehicleNumber string
	vehicleModel  string

	// License
	licenseNumber string
	licenseExpiry time.Time

	// KYC
	kycStatus    KYCStatus
	kycDocuments map[string]string // document type -> URL

	// Operational
	isAvailable       bool
	currentLocation   *valueobjects.Location
	locationUpdatedAt *time.Time

	// Performance
	completionRate   float64
	cancellationRate float64
	onTimeRate       float64

	// Timestamps
	onboardedAt *time.Time
	verifiedAt  *time.Time
}

type UserStats struct {
	TotalUsers       int64
	TotalCustomers   int64
	TotalCaptains    int64
	TotalAdmins      int64
	ActiveUsers      int64
	VerifiedCaptains int64
}
// CaptainProfile getter methods
func (cp *CaptainProfile) KYCStatus() KYCStatus {
	if cp == nil {
		return KYCPending
	}
	return cp.kycStatus
}

func (cp *CaptainProfile) IsAvailable() bool {
	if cp == nil {
		return false
	}
	return cp.isAvailable
}

func (cp *CaptainProfile) CurrentLocation() *valueobjects.Location {
	if cp == nil {
		return nil
	}
	return cp.currentLocation
}

// AdminProfile contains admin-specific data
type AdminProfile struct {
	permissions []string // List of permission codes
	department  string
	employeeID  string
}

// NewUser creates a new user
func NewUser(
	id string,
	firstName, lastName string,
	email, phone *string,
	primaryRole UserRole,
	provider string,
) (*User, error) {
	// Validation
	if id == "" {
		return nil, errors.New("user ID is required")
	}

	if firstName == "" && lastName == "" {
		return nil, errors.New("at least one name is required")
	}

	// Role-based contact validation
	switch primaryRole {
		case RoleCaptain:
			// Captains MUST have phone (for OTP login)
			if phone == nil || *phone == "" {
				return nil, errors.New("phone number is required for captains")
			}
			// Email is optional for captains
		case RoleCustomer:
			// Customers MUST have email (for Google/Apple login) OR phone
			if email == nil && phone == nil {
				return nil, errors.New("at least one contact method is required for customers")
			}
			// Note: Customers typically start with email (Google/Apple), phone is optional
		case RoleAdmin:
			// Admins MUST have email
			if email == nil || *email == "" {
				return nil, errors.New("email is required for admins")
			}
	}

	now := time.Now()

	user := &User{
		id:             id,
		version:        1,
		firstName:      firstName,
		lastName:       lastName,
		email:          email,
		phone:          phone,
		primaryRole:    primaryRole,
		roles:          []UserRole{primaryRole},
		status:         StatusActive,
		provider:       provider,
		walletBalance:  valueobjects.Zero("INR"),
		walletVersion:  1,
		walletCurrency: "INR",
		createdAt:      now,
		updatedAt:      now,
		pendingEvents:  []events.DomainEvent{},
	}

	// Initialize role-specific profiles
	if primaryRole == RoleCaptain {
		user.captainProfile = &CaptainProfile{
			kycStatus:    KYCPending,
			kycDocuments: make(map[string]string),
			isAvailable:  false,
		}
		user.status = StatusPendingKYC
	}

	if primaryRole == RoleAdmin {
		user.adminProfile = &AdminProfile{
			permissions: []string{},
		}
	}

	// Raise domain event
	user.addEvent(events.NewUserCreatedEvent(id, string(primaryRole), email, phone))

	return user, nil
}

// Getters
func (u *User) ID() string                          { return u.id }
func (u *User) Version() int                        { return u.version }
func (u *User) FirstName() string                   { return u.firstName }
func (u *User) LastName() string                    { return u.lastName }
func (u *User) FullName() string                    { return u.firstName + " " + u.lastName }
func (u *User) ProfilePic() string                  { return u.profilePic }
func (u *User) Email() *string                      { return u.email }
func (u *User) Phone() *string                      { return u.phone }
func (u *User) PhoneVerified() bool                 { return u.phoneVerified }
func (u *User) EmailVerified() bool                 { return u.emailVerified }
func (u *User) PrimaryRole() UserRole               { return u.primaryRole }
func (u *User) Roles() []UserRole                   { return u.roles }
func (u *User) Status() UserStatus                  { return u.status }
func (u *User) WalletBalance() *valueobjects.Money  { return u.walletBalance }
func (u *User) Rating() float64                     { return u.rating }
func (u *User) TotalRatings() int                   { return u.totalRatings }
func (u *User) CaptainProfile() *CaptainProfile     { return u.captainProfile }
func (u *User) AdminProfile() *AdminProfile         { return u.adminProfile }
func (u *User) CreatedAt() time.Time                { return u.createdAt }
func (u *User) UpdatedAt() time.Time                { return u.updatedAt }
func (u *User) PendingEvents() []events.DomainEvent { return u.pendingEvents }

// Business methods

// UpdateProfile updates user profile information
func (u *User) UpdateProfile(firstName, lastName, profilePic string) error {
	if firstName == "" && lastName == "" {
		return errors.New("at least one name is required")
	}

	u.firstName = firstName
	u.lastName = lastName
	u.profilePic = profilePic
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewUserProfileUpdatedEvent(u.id))

	return nil
}

// VerifyEmail marks email as verified
func (u *User) VerifyEmail() {
	u.emailVerified = true
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewUserEmailVerifiedEvent(u.id))
}

// AddPhoneNumber adds a phone number to user account (for customers who signed up with email)
func (u *User) AddPhoneNumber(phone string) error {
	if phone == "" {
		return errors.New("phone number cannot be empty")
	}

	if u.phone != nil && *u.phone != "" {
		return errors.New("phone number already exists")
	}

	u.phone = &phone
	u.phoneVerified = false // Will need to verify the new phone
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewUserPhoneAddedEvent(u.id, phone))

	return nil
}

// VerifyPhone marks phone as verified
func (u *User) VerifyPhone() {
	u.phoneVerified = true
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewUserPhoneVerifiedEvent(u.id))
}

// AddRole adds a new role to user
// A user can have multiple roles (e.g., both CUSTOMER and CAPTAIN)
func (u *User) AddRole(role UserRole) error {
	// Check if role already exists
	for _, r := range u.roles {
		if r == role {
			return errors.New("user already has this role")
		}
	}

	// Validate role-specific requirements before adding
	if role == RoleCaptain {
		// Captain role requires phone number
		if u.phone == nil || *u.phone == "" {
			return errors.New("phone number is required to become a captain")
		}
	}

	u.roles = append(u.roles, role)

	// Initialize role-specific profile if needed
	if role == RoleCaptain && u.captainProfile == nil {
		u.captainProfile = &CaptainProfile{
			kycStatus:    KYCPending,
			kycDocuments: make(map[string]string),
			isAvailable:  false,
		}
	}

	if role == RoleAdmin && u.adminProfile == nil {
		u.adminProfile = &AdminProfile{
			permissions: []string{},
		}
	}

	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewUserRoleAddedEvent(u.id, string(role)))

	return nil
}

// HasRole checks if user has a specific role
func (u *User) HasRole(role UserRole) bool {
	for _, r := range u.roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsActive checks if user is active
func (u *User) IsActive() bool {
	return u.status == StatusActive
}

// Block blocks the user
func (u *User) Block(reason string) error {
	if u.status == StatusBlocked {
		return errors.New("user is already blocked")
	}

	u.status = StatusBlocked
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewUserBlockedEvent(u.id, reason))

	return nil
}

// Unblock unblocks the user
func (u *User) Unblock() error {
	if u.status != StatusBlocked {
		return errors.New("user is not blocked")
	}

	u.status = StatusActive
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewUserUnblockedEvent(u.id))

	return nil
}

// UpdateLastActive updates the last active timestamp
func (u *User) UpdateLastActive() {
	now := time.Now()
	u.lastActiveAt = &now
	// Don't increment version or updatedAt for this lightweight operation
}

// Wallet operations

// AddToWallet adds money to wallet (with optimistic locking)
func (u *User) AddToWallet(amount *valueobjects.Money) error {
	if amount.IsNegative() {
		return errors.New("cannot add negative amount")
	}

	if amount.Currency() != u.walletCurrency {
		return errors.New("currency mismatch")
	}

	newBalance, err := u.walletBalance.Add(*amount)
	if err != nil {
		return err
	}

	oldBalance := u.walletBalance
	u.walletBalance = newBalance
	u.walletVersion++
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewWalletCreditedEvent(u.id, amount, oldBalance, newBalance))

	return nil
}

// DeductFromWallet deducts money from wallet
func (u *User) DeductFromWallet(amount *valueobjects.Money) error {
	if amount.IsNegative() {
		return errors.New("cannot deduct negative amount")
	}

	if amount.Currency() != u.walletCurrency {
		return errors.New("currency mismatch")
	}

	// Check sufficient balance
	lt, _ := u.walletBalance.LessThan(*amount)
	if lt {
		return errors.New("insufficient wallet balance")
	}

	newBalance, err := u.walletBalance.Subtract(*amount)
	if err != nil {
		return err
	}

	oldBalance := u.walletBalance
	u.walletBalance = newBalance
	u.walletVersion++
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewWalletDebitedEvent(u.id, amount, oldBalance, newBalance))

	return nil
}

// Captain-specific methods

// IsCaptain checks if user is a captain
func (u *User) IsCaptain() bool {
	return u.HasRole(RoleCaptain)
}

// SubmitKYCDocuments submits KYC documents for verification
func (u *User) SubmitKYCDocuments(documents map[string]string) error {
	if !u.IsCaptain() {
		return errors.New("only captains can submit KYC documents")
	}

	// Validate required documents
	requiredDocs := []string{"LICENSE", "VEHICLE_RC", "PROFILE_PHOTO"}
	for _, doc := range requiredDocs {
		if _, exists := documents[doc]; !exists {
			return errors.New("missing required document: " + doc)
		}
	}

	u.captainProfile.kycDocuments = documents
	u.captainProfile.kycStatus = KYCPending
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewCaptainKYCSubmittedEvent(u.id))

	return nil
}

// ApproveKyc approves captain KYC (admin action)
func (u *User) ApproveKyc() error {
	if !u.IsCaptain() {
		return errors.New("user is not a captain")
	}

	if u.captainProfile.kycStatus == KYCVerified {
		return errors.New("KYC already verified")
	}

	u.captainProfile.kycStatus = KYCVerified
	now := time.Now()
	u.captainProfile.verifiedAt = &now
	u.status = StatusActive
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewCaptainKYCApprovedEvent(u.id))

	return nil
}

// RejectKyc rejects captain KYC (admin action)
func (u *User) RejectKyc(reason string) error {
	if !u.IsCaptain() {
		return errors.New("user is not a captain")
	}

	u.captainProfile.kycStatus = KYCRejected
	u.updatedAt = time.Now()
	u.version++

	u.addEvent(events.NewCaptainKYCRejectedEvent(u.id, reason))

	return nil
}

// UpdateVehicleInfo updates vehicle information
func (u *User) UpdateVehicleInfo(vehicleType VehicleType, vehicleNumber, vehicleModel string) error {
	if !u.IsCaptain() {
		return errors.New("only captains can update vehicle info")
	}

	u.captainProfile.vehicleType = vehicleType
	u.captainProfile.vehicleNumber = vehicleNumber
	u.captainProfile.vehicleModel = vehicleModel
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// GoOnline sets captain as available
func (u *User) GoOnline(location *valueobjects.Location) error {
	if !u.IsCaptain() {
		return errors.New("only captains can go online")
	}

	if u.captainProfile.kycStatus != KYCVerified {
		return errors.New("captain must be KYC verified to go online")
	}

	if !u.IsActive() {
		return errors.New("account must be active")
	}

	u.captainProfile.isAvailable = true
	u.captainProfile.currentLocation = location
	now := time.Now()
	u.captainProfile.locationUpdatedAt = &now
	u.updatedAt = time.Now()

	u.addEvent(events.NewCaptainWentOnlineEvent(u.id, location))

	return nil
}

// GoOffline sets captain as unavailable
func (u *User) GoOffline() error {
	if !u.IsCaptain() {
		return errors.New("only captains can go offline")
	}

	u.captainProfile.isAvailable = false
	u.updatedAt = time.Now()

	u.addEvent(events.NewCaptainWentOfflineEvent(u.id))

	return nil
}

// UpdateLocation updates captain's current location
func (u *User) UpdateLocation(location *valueobjects.Location) error {
	if !u.IsCaptain() {
		return errors.New("only captains can update location")
	}

	u.captainProfile.currentLocation = location
	now := time.Now()
	u.captainProfile.locationUpdatedAt = &now
	// Don't increment version for lightweight location updates

	return nil
}

// IsAvailable checks if captain is available for orders
func (u *User) IsAvailable() bool {
	if !u.IsCaptain() {
		return false
	}

	return u.captainProfile.isAvailable &&
		u.captainProfile.kycStatus == KYCVerified &&
		u.IsActive()
}

// UpdateRating updates user's average rating
func (u *User) UpdateRating(newRating float64) error {
	if newRating < 1 || newRating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	// Calculate new average
	totalScore := u.rating * float64(u.totalRatings)
	totalScore += newRating
	u.totalRatings++
	u.rating = totalScore / float64(u.totalRatings)

	u.updatedAt = time.Now()
	u.version++

	return nil
}

// Domain event helpers

func (u *User) addEvent(event events.DomainEvent) {
	u.pendingEvents = append(u.pendingEvents, event)
}

func (u *User) ClearEvents() {
	u.pendingEvents = []events.DomainEvent{}
}
