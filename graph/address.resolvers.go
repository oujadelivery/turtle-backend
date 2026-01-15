package graph

import (
	"context"

	"turtle/graph/generated"
	"turtle/graph/model"
	"turtle/middlewares"
	"turtle/models"
	"turtle/services/address"
)

// Mutations

func (r *mutationResolver) CreateAddress(ctx context.Context, input model.CreateAddressInput) (*models.Address, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	addressInput := address.CreateAddressInput{
		UserID:       authUser.UserID,
		Label:        addressLabelToString(input.Label),
		AddressLine1: input.AddressLine1,
		AddressLine2: ptrToString(input.AddressLine2),
		Landmark:     ptrToString(input.Landmark),
		City:         input.City,
		State:        input.State,
		Country:      ptrToString(input.Country),
		PostalCode:   input.PostalCode,
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		ContactName:  ptrToString(input.ContactName),
		ContactPhone: ptrToString(input.ContactPhone),
		IsDefault:    ptrToBool(input.IsDefault),
	}

	addr, err := r.Resolver.AddressService.CreateAddress(addressInput)
	if err != nil {
		return nil, err
	}

	return addr, nil
}

// UpdateAddress is the resolver for the updateAddress field.
func (r *mutationResolver) UpdateAddress(ctx context.Context, addressID int, input model.UpdateAddressInput) (*models.Address, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	updateInput := address.UpdateAddressInput{
		Label:        stringPtr(enumToString(input.Label)),
		AddressLine1: input.AddressLine1,
		AddressLine2: input.AddressLine2,
		Landmark:     input.Landmark,
		City:         input.City,
		State:        input.State,
		PostalCode:   input.PostalCode,
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		ContactName:  input.ContactName,
		ContactPhone: input.ContactPhone,
		IsActive:     input.IsActive,
	}

	addr, err := r.Resolver.AddressService.UpdateAddress(uint(addressID), authUser.UserID, updateInput)
	if err != nil {
		return nil, err
	}

	return addr, nil
}

// DeleteAddress is the resolver for the deleteAddress field.
func (r *mutationResolver) DeleteAddress(ctx context.Context, addressID int) (*model.SuccessResponse, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if err := r.Resolver.AddressService.DeleteAddress(uint(addressID), authUser.UserID); err != nil {
		return &model.SuccessResponse{
			Success: false,
			Message: stringPtr("Failed to delete address"),
		}, err
	}

	return &model.SuccessResponse{
		Success: true,
		Message: stringPtr("Address deleted successfully"),
	}, nil
}

// SetDefaultAddress is the resolver for the setDefaultAddress field.
func (r *mutationResolver) SetDefaultAddress(ctx context.Context, addressID int) (*models.Address, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	addr, err := r.Resolver.AddressService.SetDefaultAddress(uint(addressID), authUser.UserID)
	if err != nil {
		return nil, err
	}

	return addr, nil
}

// VerifyAddress is the resolver for the verifyAddress field.
func (r *mutationResolver) VerifyAddress(ctx context.Context, addressID int) (*models.Address, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	addr, err := r.Resolver.AddressService.VerifyAddress(uint(addressID), authUser.UserID)
	if err != nil {
		return nil, err
	}

	return addr, nil
}

// Queries
// MyAddresses is the resolver for the myAddresses field.
func (r *queryResolver) MyAddresses(ctx context.Context) ([]*models.Address, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	addresses, err := r.Resolver.AddressService.GetUserAddresses(authUser.UserID)
	if err != nil {
		return nil, err
	}

	return addresses, nil
}

// Address is the resolver for the address field.
func (r *queryResolver) Address(ctx context.Context, id int) (*models.Address, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	address, err := r.Resolver.AddressService.GetAddressByID(uint(id))
	if err != nil {
		return nil, err
	}

	// Check authorization
	if address.UserID != authUser.UserID {
		return nil, middlewares.ErrUnauthorized
	}

	return address, nil
}

func (r *queryResolver) DefaultAddress(ctx context.Context) (*models.Address, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	address, err := r.Resolver.AddressService.GetDefaultAddress(authUser.UserID)
	if err != nil {
		return nil, err
	}

	return address, nil
}

func (r *queryResolver) SuggestedAddresses(ctx context.Context, timeOfDay *model.TimeOfDay) ([]*models.Address, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	timeSlot := "MORNING"
	if timeOfDay != nil {
		timeSlot = string(*timeOfDay)
	}

	addresses, err := r.Resolver.AddressService.GetSuggestedAddresses(authUser.UserID, timeSlot)
	if err != nil {
		return nil, err
	}

	return addresses, nil
}

// Field Resolvers

func (r *addressResolver) ID(ctx context.Context, obj *models.Address) (int, error) {
	return int(obj.ID), nil
}

func (r *addressResolver) UserID(ctx context.Context, obj *models.Address) (int, error) {
	return int(obj.UserID), nil
}

func (r *addressResolver) FullAddress(ctx context.Context, obj *models.Address) (string, error) {
	return obj.GetFullAddress(), nil
}

func (r *addressResolver) PreferredTimeSlot(ctx context.Context, obj *models.Address) (*model.TimeOfDay, error) {
	slot := obj.GetPreferredTimeSlot()
	timeOfDay := model.TimeOfDay(slot)
	return &timeOfDay, nil
}

// Label converts database string to GraphQL enum
// This is called automatically by GraphQL when the label field is requested
func (r *addressResolver) Label(ctx context.Context, obj *models.Address) (*model.AddressLabel, error) {
	if obj.Label == "" {
		return nil, nil
	}
	label := model.AddressLabel(obj.Label)
	return &label, nil
}

// Address returns generated.AddressResolver implementation.
func (r *Resolver) Address() generated.AddressResolver {
	return &addressResolver{r}
}

type addressResolver struct{ *Resolver }
