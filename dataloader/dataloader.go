package dataloader

import (
	"context"
	"net/http"
	"time"

	"turtle/db"
	"turtle/models"

	"github.com/graph-gophers/dataloader/v7"
	"gorm.io/gorm"
)

// Loaders contains all dataloaders
type Loaders struct {
	UserLoader          *dataloader.Loader[uint, *models.User]
	AddressLoader       *dataloader.Loader[uint, *models.Address]
	OrderLoader         *dataloader.Loader[uint, *models.Order]
	UserAddressesLoader *dataloader.Loader[uint, []*models.Address]
	OrderRatingsLoader  *dataloader.Loader[uint, []*models.Rating]
}

// contextKey is the key type for context values
type contextKey string

const loadersKey = contextKey("dataloaders")

// NewLoaders creates new dataloader instances
func NewLoaders() *Loaders {
	return &Loaders{
		UserLoader:          newUserLoader(),
		AddressLoader:       newAddressLoader(),
		OrderLoader:         newOrderLoader(),
		UserAddressesLoader: newUserAddressesLoader(),
		OrderRatingsLoader:  newOrderRatingsLoader(),
	}
}

// Middleware injects dataloaders into context
func Middleware(loaders *Loaders) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), loadersKey, loaders)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// For returns the dataloader for a given context
func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

// User loader - batches user queries
func newUserLoader() *dataloader.Loader[uint, *models.User] {
	return dataloader.NewBatchedLoader(
		func(ctx context.Context, keys []uint) []*dataloader.Result[*models.User] {
			// Batch load users by IDs
			var users []models.User
			if err := db.DB.Where("id IN ?", keys).Find(&users).Error; err != nil {
				// Return error for all keys
				results := make([]*dataloader.Result[*models.User], len(keys))
				for i := range results {
					results[i] = &dataloader.Result[*models.User]{Error: err}
				}
				return results
			}

			// Map users by ID for O(1) lookup
			userMap := make(map[uint]*models.User, len(users))
			for i := range users {
				userMap[users[i].ID] = &users[i]
			}

			// Return results in same order as keys
			results := make([]*dataloader.Result[*models.User], len(keys))
			for i, key := range keys {
				if user, ok := userMap[key]; ok {
					results[i] = &dataloader.Result[*models.User]{Data: user}
				} else {
					results[i] = &dataloader.Result[*models.User]{
						Error: gorm.ErrRecordNotFound,
					}
				}
			}
			return results
		},
		dataloader.WithWait[uint, *models.User](10*time.Millisecond),
		dataloader.WithBatchCapacity[uint, *models.User](100),
	)
}

// Address loader - batches address queries
func newAddressLoader() *dataloader.Loader[uint, *models.Address] {
	return dataloader.NewBatchedLoader(
		func(ctx context.Context, keys []uint) []*dataloader.Result[*models.Address] {
			var addresses []models.Address
			if err := db.DB.Where("id IN ?", keys).Find(&addresses).Error; err != nil {
				results := make([]*dataloader.Result[*models.Address], len(keys))
				for i := range results {
					results[i] = &dataloader.Result[*models.Address]{Error: err}
				}
				return results
			}

			addressMap := make(map[uint]*models.Address, len(addresses))
			for i := range addresses {
				addressMap[addresses[i].ID] = &addresses[i]
			}

			results := make([]*dataloader.Result[*models.Address], len(keys))
			for i, key := range keys {
				if address, ok := addressMap[key]; ok {
					results[i] = &dataloader.Result[*models.Address]{Data: address}
				} else {
					results[i] = &dataloader.Result[*models.Address]{
						Error: gorm.ErrRecordNotFound,
					}
				}
			}
			return results
		},
		dataloader.WithWait[uint, *models.Address](10*time.Millisecond),
		dataloader.WithBatchCapacity[uint, *models.Address](100),
	)
}

// Order loader - batches order queries
func newOrderLoader() *dataloader.Loader[uint, *models.Order] {
	return dataloader.NewBatchedLoader(
		func(ctx context.Context, keys []uint) []*dataloader.Result[*models.Order] {
			var orders []models.Order
			if err := db.DB.Where("id IN ?", keys).
				Preload("Customer").
				Preload("Captain").
				Find(&orders).Error; err != nil {
				results := make([]*dataloader.Result[*models.Order], len(keys))
				for i := range results {
					results[i] = &dataloader.Result[*models.Order]{Error: err}
				}
				return results
			}

			orderMap := make(map[uint]*models.Order, len(orders))
			for i := range orders {
				orderMap[orders[i].ID] = &orders[i]
			}

			results := make([]*dataloader.Result[*models.Order], len(keys))
			for i, key := range keys {
				if order, ok := orderMap[key]; ok {
					results[i] = &dataloader.Result[*models.Order]{Data: order}
				} else {
					results[i] = &dataloader.Result[*models.Order]{
						Error: gorm.ErrRecordNotFound,
					}
				}
			}
			return results
		},
		dataloader.WithWait[uint, *models.Order](10*time.Millisecond),
		dataloader.WithBatchCapacity[uint, *models.Order](100),
	)
}

// UserAddresses loader - batches queries for user's addresses
func newUserAddressesLoader() *dataloader.Loader[uint, []*models.Address] {
	return dataloader.NewBatchedLoader(
		func(ctx context.Context, userIDs []uint) []*dataloader.Result[[]*models.Address] {
			var addresses []models.Address
			if err := db.DB.Where("user_id IN ? AND is_active = true", userIDs).
				Order("is_default DESC, usage_count DESC").
				Find(&addresses).Error; err != nil {
				results := make([]*dataloader.Result[[]*models.Address], len(userIDs))
				for i := range results {
					results[i] = &dataloader.Result[[]*models.Address]{Error: err}
				}
				return results
			}

			// Group addresses by user_id
			addressesByUser := make(map[uint][]*models.Address)
			for i := range addresses {
				userID := addresses[i].UserID
				addressesByUser[userID] = append(addressesByUser[userID], &addresses[i])
			}

			// Return results in same order as keys
			results := make([]*dataloader.Result[[]*models.Address], len(userIDs))
			for i, userID := range userIDs {
				if addrs, ok := addressesByUser[userID]; ok {
					results[i] = &dataloader.Result[[]*models.Address]{Data: addrs}
				} else {
					results[i] = &dataloader.Result[[]*models.Address]{Data: []*models.Address{}}
				}
			}
			return results
		},
		dataloader.WithWait[uint, []*models.Address](10*time.Millisecond),
		dataloader.WithBatchCapacity[uint, []*models.Address](100),
	)
}

// OrderRatings loader - batches queries for order ratings
func newOrderRatingsLoader() *dataloader.Loader[uint, []*models.Rating] {
	return dataloader.NewBatchedLoader(
		func(ctx context.Context, orderIDs []uint) []*dataloader.Result[[]*models.Rating] {
			var ratings []models.Rating
			if err := db.DB.Where("order_id IN ?", orderIDs).
				Preload("Reviewer").
				Preload("ReviewedUser").
				Find(&ratings).Error; err != nil {
				results := make([]*dataloader.Result[[]*models.Rating], len(orderIDs))
				for i := range results {
					results[i] = &dataloader.Result[[]*models.Rating]{Error: err}
				}
				return results
			}

			// Group ratings by order_id
			ratingsByOrder := make(map[uint][]*models.Rating)
			for i := range ratings {
				orderID := ratings[i].OrderID
				ratingsByOrder[orderID] = append(ratingsByOrder[orderID], &ratings[i])
			}

			results := make([]*dataloader.Result[[]*models.Rating], len(orderIDs))
			for i, orderID := range orderIDs {
				if rtgs, ok := ratingsByOrder[orderID]; ok {
					results[i] = &dataloader.Result[[]*models.Rating]{Data: rtgs}
				} else {
					results[i] = &dataloader.Result[[]*models.Rating]{Data: []*models.Rating{}}
				}
			}
			return results
		},
		dataloader.WithWait[uint, []*models.Rating](10*time.Millisecond),
		dataloader.WithBatchCapacity[uint, []*models.Rating](100),
	)
}

// Example usage in resolvers:
/*
// In user.resolvers.go
func (r *userResolver) Addresses(ctx context.Context, obj *models.User) ([]*models.Address, error) {
	// This will batch all address queries within the same request
	return dataloader.For(ctx).UserAddressesLoader.Load(ctx, obj.ID)
}

// In order.resolvers.go
func (r *orderResolver) Customer(ctx context.Context, obj *models.Order) (*models.User, error) {
	// This will batch all customer queries
	return dataloader.For(ctx).UserLoader.Load(ctx, obj.CustomerID)
}

func (r *orderResolver) Captain(ctx context.Context, obj *models.Order) (*models.User, error) {
	if obj.CaptainID == nil {
		return nil, nil
	}
	return dataloader.For(ctx).UserLoader.Load(ctx, *obj.CaptainID)
}

func (r *orderResolver) PickupAddress(ctx context.Context, obj *models.Order) (*models.Address, error) {
	return dataloader.For(ctx).AddressLoader.Load(ctx, obj.PickupAddressID)
}

func (r *orderResolver) DeliveryAddress(ctx context.Context, obj *models.Order) (*models.Address, error) {
	return dataloader.For(ctx).AddressLoader.Load(ctx, obj.DeliveryAddressID)
}

// Without dataloader: N+1 query problem
// Query for 100 orders = 1 query for orders + 100 queries for customers = 101 queries

// With dataloader:
// Query for 100 orders = 1 query for orders + 1 batched query for customers = 2 queries!
*/
