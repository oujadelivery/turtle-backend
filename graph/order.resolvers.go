package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"turtle/db"
	"turtle/graph/generated"
	"turtle/graph/model"
	"turtle/middlewares"
	"turtle/models"
	"turtle/services/order"
)

// ==================== MUTATIONS ====================

// CreateOrder creates a new delivery order
func (r *mutationResolver) CreateOrder(ctx context.Context, input model.CreateOrderInput) (*models.Order, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	orderInput := order.CreateOrderInput{
		CustomerID:           authUser.UserID,
		PickupAddressID:      uint(input.PickupAddressID),
		PickupName:           input.PickupName,
		PickupPhone:          input.PickupPhone,
		PickupInstructions:   ptrToString(input.PickupInstructions),
		DeliveryAddressID:    uint(input.DeliveryAddressID),
		DeliveryName:         input.DeliveryName,
		DeliveryPhone:        input.DeliveryPhone,
		DeliveryInstructions: ptrToString(input.DeliveryInstructions),
		ParcelType:           string(input.ParcelType),
		ParcelWeight:         ptrToFloat(input.ParcelWeight),
		ParcelDescription:    ptrToString(input.ParcelDescription),
		ParcelValue:          ptrToFloat(input.ParcelValue),
		ParcelImages:         input.ParcelImages,
		PaymentMethod:        string(input.PaymentMethod),
		CouponCode:           ptrToString(input.CouponCode),
		IsPriority:           ptrToBool(input.IsPriority),
		IsInsured:            ptrToBool(input.IsInsured),
	}

	ord, err := r.Resolver.OrderService.CreateOrder(orderInput)
	if err != nil {
		return nil, err
	}

	// Publish order created event (async)
	go func() {
		r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, ord)
	}()

	return ord, nil
}

// CancelOrder cancels an order
func (r *mutationResolver) CancelOrder(ctx context.Context, orderID int, reason string) (*models.Order, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	cancelledBy := "CUSTOMER"
	if authUser.Role == RoleCaptain {
		cancelledBy = "CAPTAIN"
	}

	ord, err := r.Resolver.OrderService.CancelOrder(uint(orderID), authUser.UserID, reason, cancelledBy)
	if err != nil {
		return nil, err
	}

	// Publish order update (async)
	go r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, ord)

	return ord, nil
}

// AcceptOrder - Captain accepts order with distributed locking
func (r *mutationResolver) AcceptOrder(ctx context.Context, orderID int) (*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	// Use database transaction with row-level locking to prevent race condition
	var ord *models.Order
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the order row FOR UPDATE (prevents concurrent updates)
		var lockedOrder models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", orderID).
			First(&lockedOrder).Error; err != nil {
			return fmt.Errorf("order not found: %w", err)
		}

		// Check if order is still available
		if lockedOrder.Status != "PENDING" {
			return errors.New("order is no longer available")
		}

		if lockedOrder.CaptainID != nil {
			return errors.New("order already accepted by another captain")
		}

		// Verify captain can accept orders
		var captain models.User
		if err := tx.First(&captain, authUser.UserID).Error; err != nil {
			return err
		}

		if !captain.CanTakeOrders() {
			return errors.New("captain not eligible to accept orders")
		}

		// Accept the order
		now := time.Now()
		if err := tx.Model(&lockedOrder).Updates(map[string]interface{}{
			"captain_id":  authUser.UserID,
			"status":      "ACCEPTED",
			"accepted_at": now,
		}).Error; err != nil {
			return err
		}

		ord = &lockedOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Publish update (async)
	go r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, ord)

	return ord, nil
}

// RejectOrder - Captain rejects an order
func (r *mutationResolver) RejectOrder(ctx context.Context, orderID int, reason string) (*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	var ord models.Order
	if err := db.DB.First(&ord, orderID).Error; err != nil {
		return nil, err
	}

	// Only captain assigned to order can reject
	if ord.CaptainID == nil || *ord.CaptainID != authUser.UserID {
		return nil, errors.New("unauthorized to reject this order")
	}

	if ord.Status != "ACCEPTED" && ord.Status != "CAPTAIN_ARRIVING" {
		return nil, errors.New("order cannot be rejected at this stage")
	}

	// Reset to PENDING state
	updates := map[string]interface{}{
		"captain_id":          nil,
		"status":              "PENDING",
		"cancellation_reason": reason,
	}

	if err := db.DB.Model(&ord).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload order
	db.DB.Preload("Customer").Preload("Captain").First(&ord, orderID)

	go r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, &ord)

	return &ord, nil
}

// StartPickup - Captain starts going to pickup location
func (r *mutationResolver) StartPickup(ctx context.Context, orderID int) (*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	ord, err := r.Resolver.OrderService.StartPickup(uint(orderID), authUser.UserID)
	if err != nil {
		return nil, err
	}

	go r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, ord)

	return ord, nil
}

// ConfirmPickup - Captain confirms pickup with OTP
func (r *mutationResolver) ConfirmPickup(ctx context.Context, orderID int, otp string) (*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	ord, err := r.Resolver.OrderService.ConfirmPickup(uint(orderID), authUser.UserID, otp)
	if err != nil {
		return nil, err
	}

	go r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, ord)

	return ord, nil
}

// StartDelivery - Captain starts delivery to destination
func (r *mutationResolver) StartDelivery(ctx context.Context, orderID int) (*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	ord, err := r.Resolver.OrderService.StartDelivery(uint(orderID), authUser.UserID)
	if err != nil {
		return nil, err
	}

	go r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, ord)

	return ord, nil
}

// CompleteDelivery - Captain completes delivery with OTP
func (r *mutationResolver) CompleteDelivery(ctx context.Context, orderID int, otp string, proofPhotoURL *string) (*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	photoURL := ptrToString(proofPhotoURL)

	ord, err := r.Resolver.OrderService.CompleteDelivery(uint(orderID), authUser.UserID, otp, photoURL)
	if err != nil {
		return nil, err
	}

	go r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, ord)

	return ord, nil
}

// ScheduleOrder - Schedule order for later
func (r *mutationResolver) ScheduleOrder(ctx context.Context, input model.ScheduleOrderInput) (*models.Order, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Validate schedule time is in future
	if input.ScheduledPickupAt.Before(time.Now()) {
		return nil, errors.New("scheduled pickup time must be in the future")
	}

	// Create order as scheduled
	orderInput := order.CreateOrderInput{
		CustomerID:           authUser.UserID,
		PickupAddressID:      uint(input.OrderInput.PickupAddressID),
		PickupName:           input.OrderInput.PickupName,
		PickupPhone:          input.OrderInput.PickupPhone,
		DeliveryAddressID:    uint(input.OrderInput.DeliveryAddressID),
		DeliveryName:         input.OrderInput.DeliveryName,
		DeliveryPhone:        input.OrderInput.DeliveryPhone,
		ParcelType:           string(input.OrderInput.ParcelType),
		PaymentMethod:        string(input.OrderInput.PaymentMethod),
		IsPriority:           false,
		IsInsured:            false,
	}

	ord, err := r.Resolver.OrderService.CreateOrder(orderInput)
	if err != nil {
		return nil, err
	}

	// Update with schedule info
	db.DB.Model(ord).Updates(map[string]interface{}{
		"is_scheduled":        true,
		"scheduled_pickup_at": input.ScheduledPickupAt,
	})

	return ord, nil
}

// ReassignOrder - Admin reassigns order to different captain
func (r *mutationResolver) ReassignOrder(ctx context.Context, orderID int, captainID int) (*models.Order, error) {
	_, err := middlewares.RequireRole(ctx, RoleAdmin)
	if err != nil {
		return nil, err
	}

	var ord models.Order
	if err := db.DB.First(&ord, orderID).Error; err != nil {
		return nil, err
	}

	// Verify captain exists and can take orders
	var captain models.User
	if err := db.DB.First(&captain, captainID).Error; err != nil {
		return nil, errors.New("captain not found")
	}

	if !captain.CanTakeOrders() {
		return nil, errors.New("captain cannot accept orders")
	}

	// Reassign
	now := time.Now()
	updates := map[string]interface{}{
		"captain_id":  uint(captainID),
		"status":      "ACCEPTED",
		"accepted_at": now,
	}

	if err := db.DB.Model(&ord).Updates(updates).Error; err != nil {
		return nil, err
	}

	db.DB.Preload("Customer").Preload("Captain").First(&ord, orderID)

	go r.Resolver.RealtimeService.PublishOrderUpdate(ord.ID, ord.Status, &ord)

	return &ord, nil
}

// RefundOrder - Process refund for cancelled order
func (r *mutationResolver) RefundOrder(ctx context.Context, orderID int, reason string) (*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleAdmin)
	if err != nil {
		return nil, err
	}

	var ord models.Order
	if err := db.DB.First(&ord, orderID).Error; err != nil {
		return nil, err
	}

	if ord.Status != "CANCELLED" {
		return nil, errors.New("only cancelled orders can be refunded")
	}

	if ord.PaymentStatus != "PAID" {
		return nil, errors.New("order payment not completed")
	}

	// Calculate refund amount
	refundAmount := ord.CalculateRefundAmount()

	now := time.Now()
	updates := map[string]interface{}{
		"payment_status":            "REFUNDED",
		"refunded_at":               now,
		"refunded_by":               "ADMIN",
		"cancellation_refund_amount": refundAmount,
	}

	if err := db.DB.Model(&ord).Updates(updates).Error; err != nil {
		return nil, err
	}

	// TODO: Process actual refund via payment gateway

	_ = authUser
	db.DB.Preload("Customer").First(&ord, orderID)

	return &ord, nil
}

// ==================== QUERIES ====================

func (r *queryResolver) Order(ctx context.Context, id int) (*models.Order, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	ord, err := r.Resolver.OrderService.GetOrderByID(uint(id))
	if err != nil {
		return nil, err
	}

	// Authorization check
	if ord.CustomerID != authUser.UserID && (ord.CaptainID == nil || *ord.CaptainID != authUser.UserID) {
		// Allow admin to view any order
		if authUser.Role != RoleAdmin {
			return nil, errors.New("unauthorized")
		}
	}

	return ord, nil
}

func (r *queryResolver) OrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	ord, err := r.Resolver.OrderService.GetOrderByNumber(orderNumber)
	if err != nil {
		return nil, err
	}

	if ord.CustomerID != authUser.UserID && (ord.CaptainID == nil || *ord.CaptainID != authUser.UserID) {
		if authUser.Role != RoleAdmin {
			return nil, errors.New("unauthorized")
		}
	}

	return ord, nil
}

func (r *queryResolver) MyOrders(ctx context.Context, status *model.OrderStatus, pagination *model.PaginationInput) (*model.OrderConnection, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	page, pageSize := getPaginationParams(pagination)

	statusStr := ""
	if status != nil {
		statusStr = string(*status)
	}

	orders, total, err := r.Resolver.OrderService.GetUserOrders(authUser.UserID, statusStr, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &model.OrderConnection{
		Edges: orders,
		PageInfo: &model.PaginationInfo{
			Total:      int(total),
			Page:       page,
			PageSize:   pageSize,
			TotalPages: (int(total) + pageSize - 1) / pageSize,
			HasNext:    page*pageSize < int(total),
			HasPrev:    page > 1,
		},
	}, nil
}

func (r *queryResolver) MyDeliveries(ctx context.Context, status *model.OrderStatus, pagination *model.PaginationInput) (*model.OrderConnection, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	page, pageSize := getPaginationParams(pagination)

	statusStr := ""
	if status != nil {
		statusStr = string(*status)
	}

	orders, total, err := r.Resolver.OrderService.GetCaptainDeliveries(authUser.UserID, statusStr, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &model.OrderConnection{
		Edges: orders,
		PageInfo: &model.PaginationInfo{
			Total:      int(total),
			Page:       page,
			PageSize:   pageSize,
			TotalPages: (int(total) + pageSize - 1) / pageSize,
			HasNext:    page*pageSize < int(total),
			HasPrev:    page > 1,
		},
	}, nil
}

func (r *queryResolver) MyActiveOrder(ctx context.Context) (*models.Order, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	ord, err := r.Resolver.OrderService.GetActiveOrder(authUser.UserID)
	if err != nil {
		return nil, err
	}

	return ord, nil
}

func (r *queryResolver) MyActiveDelivery(ctx context.Context) (*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	ord, err := r.Resolver.OrderService.GetActiveDelivery(authUser.UserID)
	if err != nil {
		return nil, err
	}

	return ord, nil
}

func (r *queryResolver) NearbyOrders(ctx context.Context, latitude float64, longitude float64, radiusKm *float64) ([]*models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	radius := 10.0 // default 10km
	if radiusKm != nil {
		radius = *radiusKm
	}

	orders, err := r.Resolver.OrderService.GetNearbyOrders(latitude, longitude, radius)
	if err != nil {
		return nil, err
	}

	_ = authUser
	return orders, nil
}

func (r *queryResolver) EstimatePrice(ctx context.Context, input model.EstimatePriceInput) (*model.PriceEstimate, error) {
	distance := calculateDistance(input.PickupLat, input.PickupLng, input.DeliveryLat, input.DeliveryLng)

	estimate := r.Resolver.PricingService.CalculatePrice(
		distance,
		string(input.ParcelType),
		ptrToBool(input.IsPriority),
		ptrToString(input.CouponCode),
	)

	surgeMultiplier := 1.0
	if estimate.BasePrice+estimate.DistancePrice > 0 {
		surgeMultiplier = (estimate.SurgePrice / (estimate.BasePrice + estimate.DistancePrice)) + 1.0
	}

	return &model.PriceEstimate{
		BasePrice:         estimate.BasePrice,
		DistancePrice:     estimate.DistancePrice,
		SurgeMultiplier:   surgeMultiplier,
		SurgePrice:        estimate.SurgePrice,
		TaxAmount:         estimate.TaxAmount,
		DiscountAmount:    estimate.DiscountAmount,
		EstimatedTotal:    estimate.TotalPrice,
		EstimatedDistance: distance,
		EstimatedDuration: estimateDuration(distance),
		Currency:          "INR",
	}, nil
}

func (r *queryResolver) SearchOrders(ctx context.Context, query *string, status *model.OrderStatus, pagination *model.PaginationInput) (*model.OrderConnection, error) {
	_, err := middlewares.RequireRole(ctx, RoleAdmin)
	if err != nil {
		return nil, err
	}

	page, pageSize := getPaginationParams(pagination)

	// TODO: Implement full-text search
	// For now, just return paginated results
	statusStr := ""
	if status != nil {
		statusStr = string(*status)
	}

	var orders []*models.Order
	var total int64

	dbQuery := db.DB.Model(&models.Order{})

	if statusStr != "" {
		dbQuery = dbQuery.Where("status = ?", statusStr)
	}

	if query != nil && *query != "" {
		dbQuery = dbQuery.Where("order_number ILIKE ?", "%"+*query+"%")
	}

	dbQuery.Count(&total)

	offset := (page - 1) * pageSize
	if err := dbQuery.Preload("Customer").Preload("Captain").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&orders).Error; err != nil {
		return nil, err
	}

	return &model.OrderConnection{
		Edges: orders,
		PageInfo: &model.PaginationInfo{
			Total:      int(total),
			Page:       page,
			PageSize:   pageSize,
			TotalPages: (int(total) + pageSize - 1) / pageSize,
			HasNext:    page*pageSize < int(total),
			HasPrev:    page > 1,
		},
	}, nil
}

// ==================== SUBSCRIPTIONS ====================

func (r *subscriptionResolver) OrderUpdated(ctx context.Context, orderID int) (<-chan *models.Order, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan *models.Order, 10) // Buffered channel

	go func() {
		defer close(ch)

		channel := fmt.Sprintf("order:%d", orderID)
		eventCh, _ := r.Resolver.RealtimeService.Subscribe(ctx, channel)

		for {
			select {
			case event := <-eventCh:
				if event.Type == "order.updated" {
					ord, ok := event.Payload.(*models.Order)
					if !ok {
						continue
					}

					select {
					case ch <- ord:
					case <-ctx.Done():
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	_ = authUser
	return ch, nil
}

func (r *subscriptionResolver) CaptainLocationUpdated(ctx context.Context, orderID int) (<-chan *model.LocationUpdate, error) {
	authUser, err := middlewares.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan *model.LocationUpdate, 10)

	go func() {
		defer close(ch)

		channel := fmt.Sprintf("order:%d:location", orderID)
		eventCh, _ := r.Resolver.RealtimeService.Subscribe(ctx, channel)

		for {
			select {
			case event := <-eventCh:
				if event.Type == "location.updated" {
					payload, ok := event.Payload.(map[string]interface{})
					if !ok {
						continue
					}

					lat, _ := payload["latitude"].(float64)
					lng, _ := payload["longitude"].(float64)
					captainID := int(payload["captain_id"].(float64))

					update := &model.LocationUpdate{
						OrderID:   orderID,
						CaptainID: captainID,
						Latitude:  lat,
						Longitude: lng,
					}

					select {
					case ch <- update:
					case <-ctx.Done():
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	_ = authUser
	return ch, nil
}

// NewOrdersNearby is the resolver for the newOrdersNearby field.
func (r *subscriptionResolver) NewOrdersNearby(ctx context.Context, latitude float64, longitude float64, radiusKm float64) (<-chan *models.Order, error) {
	authUser, err := middlewares.RequireRole(ctx, RoleCaptain)
	if err != nil {
		return nil, err
	}

	ch := make(chan *models.Order, 10)

	go func() {
		defer close(ch)

		// Subscribe to new orders in the area
		channel := fmt.Sprintf("captain:%d:nearby_orders", authUser.UserID)
		eventCh, _ := r.Resolver.RealtimeService.Subscribe(ctx, channel)

		for {
			select {
			case event := <-eventCh:
				if event.Type == "order.created" {
					order, ok := event.Payload.(*models.Order)
					if !ok {
						continue
					}

					// Calculate distance
					dist := calculateDistance(latitude, longitude, order.PickupLat, order.PickupLng)
					if dist <= radiusKm {
						select {
						case ch <- order:
						case <-ctx.Done():
							return
						}
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// ==================== FIELD RESOLVERS ====================

func (r *orderResolver) ID(ctx context.Context, obj *models.Order) (int, error) {
	return int(obj.ID), nil
}

func (r *orderResolver) CustomerID(ctx context.Context, obj *models.Order) (int, error) {
	return int(obj.CustomerID), nil
}

func (r *orderResolver) CaptainID(ctx context.Context, obj *models.Order) (*int, error) {
	if obj.CaptainID == nil {
		return nil, nil
	}
	id := int(*obj.CaptainID)
	return &id, nil
}

func (r *orderResolver) PickupAddress(ctx context.Context, obj *models.Order) (*models.Address, error) {
	// Load if not preloaded
	if obj.PickupAddressID == 0 {
		return nil, nil
	}

	var addr models.Address
	if err := db.DB.First(&addr, obj.PickupAddressID).Error; err != nil {
		return nil, err
	}

	return &addr, nil
}

func (r *orderResolver) DeliveryAddress(ctx context.Context, obj *models.Order) (*models.Address, error) {
	if obj.DeliveryAddressID == 0 {
		return nil, nil
	}

	var addr models.Address
	if err := db.DB.First(&addr, obj.DeliveryAddressID).Error; err != nil {
		return nil, err
	}

	return &addr, nil
}

func (r *orderResolver) ParcelType(ctx context.Context, obj *models.Order) (model.ParcelType, error) {
	return model.ParcelType(obj.ParcelType), nil
}

func (r *orderResolver) ParcelImages(ctx context.Context, obj *models.Order) ([]string, error) {
	if obj.ParcelImages == "" {
		return []string{}, nil
	}

	var images []string
	if err := json.Unmarshal([]byte(obj.ParcelImages), &images); err != nil {
		return []string{}, nil
	}

	return images, nil
}

func (r *orderResolver) Status(ctx context.Context, obj *models.Order) (model.OrderStatus, error) {
	return model.OrderStatus(obj.Status), nil
}

func (r *orderResolver) StatusHistory(ctx context.Context, obj *models.Order) ([]*model.StatusHistoryItem, error) {
	if obj.StatusHistory == "" {
		return []*model.StatusHistoryItem{}, nil
	}

	var history []map[string]interface{}
	if err := json.Unmarshal([]byte(obj.StatusHistory), &history); err != nil {
		return []*model.StatusHistoryItem{}, nil
	}

	items := make([]*model.StatusHistoryItem, 0, len(history))
	for _, h := range history {
		status, _ := h["status"].(string)
		note, _ := h["note"].(string)
		timestamp, _ := h["timestamp"].(string)

		t, _ := time.Parse(time.RFC3339, timestamp)

		items = append(items, &model.StatusHistoryItem{
			Status:    model.OrderStatus(status),
			Note:      optStringPtr(note),
			Timestamp: t,
		})
	}

	return items, nil
}

func (r *orderResolver) PaymentMethod(ctx context.Context, obj *models.Order) (model.PaymentMethod, error) {
	return model.PaymentMethod(obj.PaymentMethod), nil
}

func (r *orderResolver) PaymentStatus(ctx context.Context, obj *models.Order) (model.PaymentStatus, error) {
	return model.PaymentStatus(obj.PaymentStatus), nil
}

func (r *orderResolver) CurrentLocation(ctx context.Context, obj *models.Order) (*model.Location, error) {
	// Get latest location for this order
	if obj.CaptainID == nil {
		return nil, nil
	}

	var captain models.User
	if err := db.DB.First(&captain, *obj.CaptainID).Error; err != nil {
		return nil, nil
	}

	if captain.CurrentLat == 0 && captain.CurrentLng == 0 {
		return nil, nil
	}

	return &model.Location{
		Latitude:  captain.CurrentLat,
		Longitude: captain.CurrentLng,
	}, nil
}

func (r *orderResolver) Customer(ctx context.Context, obj *models.Order) (*models.User, error) {
	return &obj.Customer, nil
}

func (r *orderResolver) Captain(ctx context.Context, obj *models.Order) (*models.User, error) {
	return obj.Captain, nil
}

func (r *orderResolver) CanCancel(ctx context.Context, obj *models.Order) (bool, error) {
	return obj.CanCancel(), nil
}

func (r *orderResolver) IsCompleted(ctx context.Context, obj *models.Order) (bool, error) {
	return obj.IsCompleted(), nil
}

// Order returns generated.OrderResolver implementation.
func (r *Resolver) Order() generated.OrderResolver { return &orderResolver{r} }

type orderResolver struct{ *Resolver }
