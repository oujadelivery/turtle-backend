package postgres

import (
	"fmt"

	"turtle/internal/domain/aggregates"
	"turtle/internal/domain/valueobjects"

	"github.com/lib/pq"
)

// ============================================================================
// MAPPERS: Domain Model ↔ Database Model
// These functions maintain separation between domain and persistence layers
// ============================================================================

// UserToDomain converts database UserModel to domain User aggregate
// Note: This uses a reconstruction pattern that bypasses domain validations
// for loading existing entities from the database
func UserToDomain(model *UserModel) (*aggregates.User, error) {
	if model == nil {
		return nil, nil
	}

	// Create user with basic info
	user, err := aggregates.NewUser(
		model.ID,
		model.FirstName,
		model.LastName,
		model.Email,
		model.Phone,
		aggregates.UserRole(model.PrimaryRole),
		model.Provider,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create domain user: %w", err)
	}

	// Restore verification status
	if model.EmailVerified && !user.EmailVerified() {
		user.VerifyEmail()
	}
	if model.PhoneVerified && !user.PhoneVerified() {
		user.VerifyPhone()
	}

	// Restore profile picture
	if model.ProfilePic != "" {
		user.UpdateProfile(model.FirstName, model.LastName, model.ProfilePic)
	}

	// Restore additional roles
	for _, roleStr := range model.Roles {
		role := aggregates.UserRole(roleStr)
		if !user.HasRole(role) {
			// Add role without domain validation for reconstruction
			user.AddRole(role)
		}
	}

	// Note: Wallet balance, ratings, and other stats are managed through
	// specific repository methods (UpdateWallet, UpdateRating, etc.)
	// When loading from DB, these values are already persisted and don't need
	// to be "restored" through domain methods. The repository reads them directly.

	// Captain and Admin profiles are loaded separately via GORM preloading
	// and their data is accessed through the respective profile models

	// Clear any domain events raised during reconstruction
	// Events should only fire for new operations, not when loading from DB
	user.ClearEvents()

	return user, nil
}

// DomainToUser converts domain User aggregate to database UserModel
func DomainToUser(user *aggregates.User) (*UserModel, error) {
	if user == nil {
		return nil, nil
	}

	// Convert roles to string array
	roles := make([]string, len(user.Roles()))
	for i, role := range user.Roles() {
		roles[i] = string(role)
	}

	model := &UserModel{
		ID:             user.ID(),
		Version:        user.Version(),
		FirstName:      user.FirstName(),
		LastName:       user.LastName(),
		ProfilePic:     user.ProfilePic(),
		Email:          user.Email(),
		EmailVerified:  user.EmailVerified(),
		Phone:          user.Phone(),
		PhoneVerified:  user.PhoneVerified(),
		PrimaryRole:    string(user.PrimaryRole()),
		Roles:          pq.StringArray(roles),
		Status:         string(user.Status()),
		WalletBalance:  user.WalletBalance().Amount(),
		WalletCurrency: user.WalletBalance().Currency(),
		CreatedAt:      user.CreatedAt(),
		UpdatedAt:      user.UpdatedAt(),
	}

	// Handle captain profile
	if user.CaptainProfile() != nil {
		model.CaptainProfile = DomainToCaptainProfile(user.ID(), user.CaptainProfile())
	}

	// Handle admin profile
	if user.AdminProfile() != nil {
		model.AdminProfile = DomainToAdminProfile(user.ID(), user.AdminProfile())
	}

	return model, nil
}

// DomainToCaptainProfile converts domain CaptainProfile to database model
func DomainToCaptainProfile(userID string, profile *aggregates.CaptainProfile) *CaptainProfileModel {
	if profile == nil {
		return nil
	}

	model := &CaptainProfileModel{
		UserID:    userID,
		KYCStatus: string(profile.KYCStatus()),
	}

	// Set location if available
	if loc := profile.CurrentLocation(); loc != nil {
		// Convert Location to PostGIS POINT
		locationStr := fmt.Sprintf("POINT(%f %f)", loc.Longitude(), loc.Latitude())
		model.CurrentLocation = &locationStr
	}

	return model
}

// DomainToAdminProfile converts domain AdminProfile to database model
func DomainToAdminProfile(userID string, profile *aggregates.AdminProfile) *AdminProfileModel {
	if profile == nil {
		return nil
	}

	return &AdminProfileModel{
		UserID: userID,
	}
}

// AddressToDomain converts database AddressModel to domain Address aggregate
func AddressToDomain(model *AddressModel) (*aggregates.Address, error) {
	if model == nil {
		return nil, nil
	}

	// Parse location from PostGIS format
	location, err := parsePostGISPoint(model.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to parse location: %w", err)
	}

	// Get label
	var label aggregates.AddressLabel
	if model.Label != nil {
		label = aggregates.AddressLabel(*model.Label)
	} else {
		label = aggregates.AddressLabelOther
	}

	// Create address
	addressLine2 := ""
	if model.AddressLine2 != nil {
		addressLine2 = *model.AddressLine2
	}

	landmark := ""
	if model.Landmark != nil {
		landmark = *model.Landmark
	}

	postalCode := ""
	if model.PostalCode != nil {
		postalCode = *model.PostalCode
	}

	contactName := ""
	if model.ContactName != nil {
		contactName = *model.ContactName
	}

	contactPhone := ""
	if model.ContactPhone != nil {
		contactPhone = *model.ContactPhone
	}

	address, err := aggregates.NewAddress(
		model.ID,
		model.UserID,
		label,
		model.AddressLine1,
		addressLine2,
		landmark,
		model.City,
		model.State,
		model.Country,
		postalCode,
		location,
		contactName,
		contactPhone,
	)
	if err != nil {
		return nil, err
	}

	// Restore state
	if model.IsDefault {
		address.SetAsDefault()
	}
	if model.IsVerified {
		address.MarkAsVerified()
	}

	// Note: Analytics fields would need to be restored via reflection or setters

	return address, nil
}

// DomainToAddress converts domain Address aggregate to database AddressModel
func DomainToAddress(address *aggregates.Address) (*AddressModel, error) {
	if address == nil {
		return nil, nil
	}

	// Convert location to PostGIS format
	locationStr := fmt.Sprintf("POINT(%f %f)",
		address.Location().Longitude(),
		address.Location().Latitude(),
	)

	label := string(address.Label())

	// Convert monthly usage to JSONB
	monthlyUsage := make(map[string]interface{})
	for month, count := range address.MonthlyUsageCount() {
		monthlyUsage[fmt.Sprintf("%d", month)] = count
	}

	model := &AddressModel{
		ID:                  address.ID(),
		UserID:              address.UserID(),
		Version:             address.Version(),
		Label:               &label,
		AddressLine1:        address.AddressLine1(),
		City:                address.City(),
		State:               address.State(),
		Country:             address.Country(),
		Location:            locationStr,
		IsDefault:           address.IsDefault(),
		IsActive:            address.IsActive(),
		IsVerified:          address.IsVerified(),
		UsageCount:          address.UsageCount(),
		LastUsedAt:          address.LastUsedAt(),
		MorningUsageCount:   address.MorningUsageCount(),
		AfternoonUsageCount: address.AfternoonUsageCount(),
		EveningUsageCount:   address.EveningUsageCount(),
		NightUsageCount:     address.NightUsageCount(),
		WeekdayUsageCount:   address.WeekdayUsageCount(),
		WeekendUsageCount:   address.WeekendUsageCount(),
		MonthlyUsageCount:   monthlyUsage,
		CreatedAt:           address.CreatedAt(),
		UpdatedAt:           address.UpdatedAt(),
	}

	// Set optional fields
	if address.AddressLine2() != "" {
		line2 := address.AddressLine2()
		model.AddressLine2 = &line2
	}
	if address.Landmark() != "" {
		landmark := address.Landmark()
		model.Landmark = &landmark
	}
	if address.PostalCode() != "" {
		postalCode := address.PostalCode()
		model.PostalCode = &postalCode
	}
	if address.ContactName() != "" {
		contactName := address.ContactName()
		model.ContactName = &contactName
	}
	if address.ContactPhone() != "" {
		contactPhone := address.ContactPhone()
		model.ContactPhone = &contactPhone
	}

	return model, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// parsePostGISPoint parses a PostGIS POINT to Location
// Handles both WKT text format and returns from ST_AsText
func parsePostGISPoint(pointStr string) (*valueobjects.Location, error) {
	if pointStr == "" {
		return nil, fmt.Errorf("empty point string")
	}

	// Try parsing "POINT(lng lat)" format (Well-Known Text)
	var lng, lat float64
	n, err := fmt.Sscanf(pointStr, "POINT(%f %f)", &lng, &lat)
	if err == nil && n == 2 {
		return valueobjects.NewLocation(lat, lng)
	}

	// Try alternative formats
	// "POINT (lng lat)" with space after POINT
	n, err = fmt.Sscanf(pointStr, "POINT (%f %f)", &lng, &lat)
	if err == nil && n == 2 {
		return valueobjects.NewLocation(lat, lng)
	}

	return nil, fmt.Errorf("unsupported point format: %s", pointStr)
}

// formatPostGISPoint formats a Location to PostGIS POINT string
func formatPostGISPoint(loc *valueobjects.Location) string {
	if loc == nil {
		return ""
	}
	return fmt.Sprintf("POINT(%f %f)", loc.Longitude(), loc.Latitude())
}

// parseJSONB parses JSONB to map
func parseJSONB(data JSONB) map[int]int {
	result := make(map[int]int)
	for k, v := range data {
		var month int
		fmt.Sscanf(k, "%d", &month)
		if count, ok := v.(float64); ok {
			result[month] = int(count)
		}
	}
	return result
}

// toJSONB converts map to JSONB
func toJSONB(data map[int]int) JSONB {
	result := make(map[string]interface{})
	for k, v := range data {
		result[fmt.Sprintf("%d", k)] = v
	}
	return result
}

// stringArrayToSlice converts pq.StringArray to []string
func stringArrayToSlice(arr pq.StringArray) []string {
	return []string(arr)
}

// sliceToStringArray converts []string to pq.StringArray
func sliceToStringArray(slice []string) pq.StringArray {
	return pq.StringArray(slice)
}

// parseKYCDocuments parses KYC documents from JSONB
func parseKYCDocuments(data JSONB) map[string]string {
	result := make(map[string]string)

	for key, value := range data {
		if strValue, ok := value.(string); ok {
			result[key] = strValue
		}
	}

	return result
}

// formatKYCDocuments formats KYC documents to JSONB
func formatKYCDocuments(docs map[string]string) JSONB {
	if len(docs) == 0 {
		return make(JSONB)
	}

	result := make(JSONB)
	for key, value := range docs {
		result[key] = value
	}

	return result
}
