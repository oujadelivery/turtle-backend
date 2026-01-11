package graph

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
	"turtle/graph/generated"
	"turtle/graph/model"
	"turtle/pkg/jwt"
	"turtle/services/auth"
)

type Resolver struct{}

// SocialLogin is the resolver for the socialLogin field.
func (r *mutationResolver) SocialLogin(ctx context.Context, provider model.Provider, providerToken string) (*model.AuthPayload, error) {
	user := auth.SocialLogin(provider.String(), providerToken)
	token, _ := jwt.GenerateToken(user.ID)
	return &model.AuthPayload{Token: token}, nil
}

func (r *queryResolver) Health(ctx context.Context) (string, error) {
	return "ok", nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
/*
	type Resolver struct{}
*/
