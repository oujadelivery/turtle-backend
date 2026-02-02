package graph

import (
	"turtle/graph/model"
	"turtle/internal/domain/aggregates"
	"turtle/internal/domain/valueobjects"
)

// usersToGraphQL converts slice of domain Users to GraphQL Users
func usersToGraphQL(users []*aggregates.User) []*model.User {
	result := make([]*model.User, len(users))
	for i, user := range users {
		result[i] = userToGraphQL(user)
	}
	return result
}

// captainProfileToGraphQL converts domain CaptainProfile to GraphQL CaptainProfile
func captainProfileToGraphQL(profile *aggregates.CaptainProfile) *model.CaptainProfile {
	if profile == nil {
		return nil
	}

	gqlProfile := &model.CaptainProfile{
		KycStatus:   model.KYCStatus(profile.KYCStatus()),
		IsAvailable: profile.IsAvailable(),
	}

	// Convert current location if available
	if loc := profile.CurrentLocation(); loc != nil {
		gqlProfile.CurrentLocation = &model.Location{
			Latitude:  loc.Latitude(),
			Longitude: loc.Longitude(),
		}
	}

	return gqlProfile
}

// ============================================================================
// ADDRESS MAPPERS
// ============================================================================

// addressToGraphQL converts domain Address to GraphQL Address
func addressToGraphQL(address *aggregates.Address) *model.Address {
	if address == nil {
		return nil
	}

	gqlAddress := &model.Address{
		ID:           address.ID(),
		UserID:       address.UserID(),
		Label:        model.AddressLabel(address.Label()),
		AddressLine1: address.AddressLine1(),
		AddressLine2: stringPtr(address.AddressLine2()),
		Landmark:     stringPtr(address.Landmark()),
		City:         address.City(),
		State:        address.State(),
		Country:      address.Country(),
		PostalCode:   stringPtr(address.PostalCode()),
		Location: &model.Location{
			Latitude:  address.Location().Latitude(),
			Longitude: address.Location().Longitude(),
		},
		ContactName:  stringPtr(address.ContactName()),
		ContactPhone: stringPtr(address.ContactPhone()),
		IsDefault:    address.IsDefault(),
		IsActive:     address.IsActive(),
		IsVerified:   address.IsVerified(),
		UsageCount:   address.UsageCount(),
		CreatedAt:    address.CreatedAt(),
		UpdatedAt:    address.UpdatedAt(),
	}

	if lastUsed := address.LastUsedAt(); lastUsed != nil {
		gqlAddress.LastUsedAt = lastUsed
	}

	return gqlAddress
}

// addressesToGraphQL converts slice of domain Addresses to GraphQL Addresses
func addressesToGraphQL(addresses []*aggregates.Address) []*model.Address {
	result := make([]*model.Address, len(addresses))
	for i, address := range addresses {
		result[i] = addressToGraphQL(address)
	}
	return result
}

// addressLabelToDomain converts GraphQL AddressLabel to domain AddressLabel
func addressLabelToDomain(label model.AddressLabel) aggregates.AddressLabel {
	switch label {
	case model.AddressLabelHome:
		return aggregates.AddressLabelHome
	case model.AddressLabelWork:
		return aggregates.AddressLabelWork
	case model.AddressLabelOther:
		return aggregates.AddressLabelOther
	default:
		return aggregates.AddressLabelOther
	}
}

func userToGraphQL(user *aggregates.User) *model.User {
	if user == nil {
		return nil
	}

	return &model.User{
		ID:              user.ID(),
		FirstName:       user.FirstName(),
		LastName:        user.LastName(),
		FullName:        user.FullName(),
		ProfilePic:      stringPtr(user.ProfilePic()),
		Email:           user.Email(),
		EmailVerified:   user.EmailVerified(),
		Phone:           user.Phone(),
		PhoneVerified:   user.PhoneVerified(),
		PrimaryRole:     model.UserRole(user.PrimaryRole()),
		Roles:           rolesToGraphQL(user.Roles()),
		Status:          model.UserStatus(user.Status()),
		WalletBalance:   moneyToGraphQL(user.WalletBalance()),
		Rating:          floatPtr(user.Rating()),
		TotalRatings:    user.TotalRatings(),
		TotalOrders:     0, // TODO: Get from user
		TotalDeliveries: 0, // TODO: Get from user
		CaptainProfile:  captainProfileToGraphQL(user.CaptainProfile()),
		CreatedAt:       user.CreatedAt(),
		UpdatedAt:       user.UpdatedAt(),
	}
}
func rolesToGraphQL(roles []aggregates.UserRole) []model.UserRole {
	result := make([]model.UserRole, len(roles))
	for i, role := range roles {
		result[i] = model.UserRole(role)
	}
	return result
}
func moneyToGraphQL(money *valueobjects.Money) *model.Money {
	if money == nil {
		return nil
	}

	return &model.Money{
		Amount:        int(money.Amount()),
		Currency:      money.Currency(),
		DisplayAmount: money.AmountInMajorUnit(),
		Formatted:     money.Format("symbol"),
	}
}