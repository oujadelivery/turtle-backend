package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"turtle/db"
	"turtle/infra"
	"turtle/models"
	"turtle/services/pricing"
)

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidStatus       = errors.New("invalid order status")
	ErrCaptainNotAvailable = errors.New("captain not available")
	ErrDuplicateOrder      = errors.New("duplicate order detected")
	ErrOrderLocked         = errors.New("order is being processed by another request")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrConcurrentUpdate    = errors.New("order was modified by another process")
)

type Service struct {
	pricingService *pricing.Service
}

func NewService() *Service {
	return &Service{
		pricingService: pricing.NewService(),
	}
}

type CreateOrderInput struct {
	CustomerID           uint
	PickupAddressID      uint
	PickupName           string
	PickupPhone          string
	PickupInstructions   string
	DeliveryAddressID    uint
	DeliveryName         string
	DeliveryPhone        string
	DeliveryInstructions string
	ParcelType           string
	ParcelWeight         float64
	ParcelDescription    string
	ParcelValue          float64
	ParcelImages         []string
	PaymentMethod        string
	CouponCode           string
	IsPriority           bool
	IsInsured            bool
	IdempotencyKey       string // CRITICAL: Prevent duplicate orders
}

// IMPROVEMENT: Idempotency handling for order creation
func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (*models.Order, error) {
	// Validate idempotency key
	if input.IdempotencyKey == "" {
		input.IdempotencyKey = uuid.New().String()
	}

	// Check for duplicate order using idempotency key
	idempotencyLock := fmt.Sprintf("idempotency:%s", input.IdempotencyKey)
	locked, err := infra.Redis.SetNX(ctx, idempotencyLock, "1", 10*time.Minute).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire idempotency lock: %w", err)
	}
	if !locked {
		// Check if order already exists
		var existingOrder models.Order
		if err := db.DB.Where("idempotency_key = ?", input.IdempotencyKey).First(&existingOrder).Error; err == nil {
			return &existingOrder, nil // Return existing order
		}
		return nil, ErrDuplicateOrder
	}
	defer infra.Redis.Del(ctx, idempotencyLock)

	// Check for active pending orders (prevent multiple active orders)
	var activeCount int64
	db.DB.Model(&models.Order{}).
		Where("customer_id = ? AND status NOT IN ('DELIVERED', 'CANCELLED')", input.CustomerID).
		Count(&activeCount)

	if activeCount >= 3 { // Max 3 active orders
		return nil, errors.New("maximum active orders limit reached")
	}

	// Get addresses with row locking to prevent concurrent modifications
	var pickupAddr, deliveryAddr models.Address
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&pickupAddr, input.PickupAddressID).Error; err != nil {
			return errors.New("pickup address not found")
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&deliveryAddr, input.DeliveryAddressID).Error; err != nil {
			return errors.New("delivery address not found")
		}

		// Validate addresses belong to customer
		if pickupAddr.UserID != input.CustomerID || deliveryAddr.UserID != input.CustomerID {
			return errors.New("address does not belong to customer")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Calculate distance and pricing
	distance := calculateDistance(pickupAddr.Latitude, pickupAddr.Longitude,
		deliveryAddr.Latitude, deliveryAddr.Longitude)

	// Validate distance (prevent extreme values)
	if distance < 0.1 || distance > 500 { // 100m to 500km
		return nil, errors.New("invalid delivery distance")
	}

	duration := estimateDuration(distance)
	priceEstimate := s.pricingService.CalculatePrice(distance, input.ParcelType,
		input.IsPriority, input.CouponCode)

	// Generate secure OTPs using crypto/rand
	pickupOTP := generateSecureOTP()
	deliveryOTP := generateSecureOTP()
	orderNumber := generateOrderNumber()

	// Create order within transaction
	var order *models.Order
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		order = &models.Order{
			OrderNumber:          orderNumber,
			IdempotencyKey:       input.IdempotencyKey,
			CustomerID:           input.CustomerID,
			PickupAddressID:      input.PickupAddressID,
			PickupName:           input.PickupName,
			PickupPhone:          input.PickupPhone,
			PickupLat:            pickupAddr.Latitude,
			PickupLng:            pickupAddr.Longitude,
			PickupAddress:        pickupAddr.GetFullAddress(),
			PickupInstructions:   input.PickupInstructions,
			DeliveryAddressID:    input.DeliveryAddressID,
			DeliveryName:         input.DeliveryName,
			DeliveryPhone:        input.DeliveryPhone,
			DeliveryLat:          deliveryAddr.Latitude,
			DeliveryLng:          deliveryAddr.Longitude,
			DeliveryAddress:      deliveryAddr.GetFullAddress(),
			DeliveryInstructions: input.DeliveryInstructions,
			ParcelType:           input.ParcelType,
			ParcelWeight:         input.ParcelWeight,
			ParcelDescription:    input.ParcelDescription,
			ParcelValue:          input.ParcelValue,
			EstimatedDistance:    distance,
			EstimatedDuration:    duration,
			BasePrice:            priceEstimate.BasePrice,
			DistancePrice:        priceEstimate.DistancePrice,
			SurgePrice:           priceEstimate.SurgePrice,
			DiscountAmount:       priceEstimate.DiscountAmount,
			TaxAmount:            priceEstimate.TaxAmount,
			TotalPrice:           priceEstimate.TotalPrice,
			Currency:             "INR",
			CouponCode:           input.CouponCode,
			PromoDiscount:        priceEstimate.DiscountAmount,
			Status:               "PENDING",
			PlacedAt:             time.Now(),
			PickupOTP:            pickupOTP,
			DeliveryOTP:          deliveryOTP,
			PaymentMethod:        input.PaymentMethod,
			PaymentStatus:        "PENDING",
			IsPriority:           input.IsPriority,
			IsInsured:            input.IsInsured,
			Version:              1, // Optimistic locking
		}

		if len(input.ParcelImages) > 0 {
			imagesJSON, _ := json.Marshal(input.ParcelImages)
			order.ParcelImages = string(imagesJSON)
		}

		if input.IsInsured && input.ParcelValue > 0 {
			order.InsuranceAmount = input.ParcelValue * 0.02
			order.TotalPrice += order.InsuranceAmount
		}

		order.CalculateCaptainEarnings(20) // 20% platform commission
		order.StatusHistory = s.createStatusHistory("PENDING", "Order placed")

		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		// Update address usage statistics
		if err := tx.Model(&pickupAddr).Updates(map[string]interface{}{
			"usage_count":  gorm.Expr("usage_count + 1"),
			"last_used_at": time.Now(),
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&deliveryAddr).Updates(map[string]interface{}{
			"usage_count":  gorm.Expr("usage_count + 1"),
			"last_used_at": time.Now(),
		}).Error; err != nil {
			return err
		}

		// Increment coupon usage if applicable
		if input.CouponCode != "" {
			if err := tx.Model(&models.Coupon{}).
				Where("code = ?", input.CouponCode).
				Update("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Async: Notify nearby captains
	go s.notifyNearbyCaptains(ctx, order)

	// Async: Send confirmation to customer
	go s.sendOrderConfirmation(ctx, order)

	return order, nil
}

// GetOrderByID retrieves order by ID
func (s *Service) GetOrderByID(orderID uint) (*models.Order, error) {
	var order models.Order
	if err := db.DB.Preload("Customer").Preload("Captain").Preload("PickupAddress").Preload("DeliveryAddress").First(&order, orderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

// GetOrderByNumber retrieves order by order number
func (s *Service) GetOrderByNumber(orderNumber string) (*models.Order, error) {
	var order models.Order
	if err := db.DB.Preload("Customer").Preload("Captain").Where("order_number = ?", orderNumber).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

// GetUserOrders retrieves orders for a user
func (s *Service) GetUserOrders(userID uint, status string, page, pageSize int) ([]*models.Order, int64, error) {
	var orders []*models.Order
	var total int64

	query := db.DB.Model(&models.Order{}).Where("customer_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Preload("Captain").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetCaptainDeliveries retrieves deliveries for a captain
func (s *Service) GetCaptainDeliveries(captainID uint, status string, page, pageSize int) ([]*models.Order, int64, error) {
	var orders []*models.Order
	var total int64

	query := db.DB.Model(&models.Order{}).Where("captain_id = ?", captainID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Preload("Customer").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetActiveOrder retrieves active order for customer
func (s *Service) GetActiveOrder(customerID uint) (*models.Order, error) {
	var order models.Order
	if err := db.DB.Preload("Captain").Where("customer_id = ? AND status NOT IN ('DELIVERED', 'CANCELLED')", customerID).Order("created_at DESC").First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No active order
		}
		return nil, err
	}
	return &order, nil
}

// GetActiveDelivery retrieves active delivery for captain
func (s *Service) GetActiveDelivery(captainID uint) (*models.Order, error) {
	var order models.Order
	if err := db.DB.Preload("Customer").Where("captain_id = ? AND status NOT IN ('DELIVERED', 'CANCELLED')", captainID).Order("accepted_at DESC").First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetNearbyOrders retrieves pending orders near captain location
func (s *Service) GetNearbyOrders(lat, lng, radiusKm float64) ([]*models.Order, error) {
	var orders []*models.Order

	query := `
        SELECT * FROM orders 
        WHERE status = 'PENDING'
        AND captain_id IS NULL
        AND (
            6371 * acos(
                cos(radians(?)) * cos(radians(pickup_lat)) *
                cos(radians(pickup_lng) - radians(?)) +
                sin(radians(?)) * sin(radians(pickup_lat))
            )
        ) <= ?
        ORDER BY created_at DESC
        LIMIT 20
    `

	if err := db.DB.Raw(query, lat, lng, lat, radiusKm).Scan(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

// IMPROVEMENT: AcceptOrder with distributed lock and optimistic locking
func (s *Service) AcceptOrder(ctx context.Context, orderID, captainID uint) (*models.Order, error) {
	lockKey := fmt.Sprintf("order:accept:%d", orderID)
	lockValue := fmt.Sprintf("%d:%d", captainID, time.Now().Unix())

	// Try to acquire distributed lock with 10 second timeout
	locked, err := infra.Redis.SetNX(ctx, lockKey, lockValue, 10*time.Second).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return nil, ErrOrderLocked
	}
	defer infra.Redis.Del(ctx, lockKey)

	var order *models.Order
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the order row with optimistic locking
		var lockedOrder models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", orderID).
			First(&lockedOrder).Error; err != nil {
			return ErrOrderNotFound
		}

		// Validate order state
		if lockedOrder.Status != "PENDING" {
			return errors.New("order is not available for acceptance")
		}

		if lockedOrder.CaptainID != nil {
			return errors.New("order already assigned to another captain")
		}

		// Check if order is expired (e.g., 30 minutes old)
		if time.Since(lockedOrder.PlacedAt) > 30*time.Minute {
			// Auto-cancel expired pending orders
			tx.Model(&lockedOrder).Updates(map[string]interface{}{
				"status":              "CANCELLED",
				"cancelled_at":        time.Now(),
				"cancellation_reason": "Order expired - no captain available",
				"cancelled_by":        "SYSTEM",
			})
			return errors.New("order has expired")
		}

		// Verify captain eligibility
		var captain models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&captain, captainID).Error; err != nil {
			return errors.New("captain not found")
		}

		if !captain.CanTakeOrders() {
			return ErrCaptainNotAvailable
		}

		// Check captain's current active deliveries
		var activeDeliveries int64
		tx.Model(&models.Order{}).
			Where("captain_id = ? AND status NOT IN ('DELIVERED', 'CANCELLED')", captainID).
			Count(&activeDeliveries)

		if activeDeliveries >= 3 { // Max 3 concurrent deliveries
			return errors.New("captain has reached maximum concurrent deliveries")
		}

		// Calculate distance from captain to pickup
		distanceToCaptain := calculateDistance(
			captain.CurrentLat, captain.CurrentLng,
			lockedOrder.PickupLat, lockedOrder.PickupLng,
		)

		// Validate captain is within reasonable distance (50km)
		if distanceToCaptain > 50 {
			return errors.New("captain is too far from pickup location")
		}

		now := time.Now()
		estimatedArrival := now.Add(time.Duration(distanceToCaptain*2) * time.Minute)

		// Update order with optimistic locking
		result := tx.Model(&lockedOrder).
			Where("version = ?", lockedOrder.Version).
			Updates(map[string]interface{}{
				"captain_id":             captainID,
				"status":                 "ACCEPTED",
				"accepted_at":            now,
				"estimated_arrival_time": estimatedArrival,
				"version":                gorm.Expr("version + 1"),
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return ErrConcurrentUpdate
		}

		// Update status history
		lockedOrder.StatusHistory = s.addToStatusHistory(
			lockedOrder.StatusHistory,
			"ACCEPTED",
			fmt.Sprintf("Captain accepted order. ETA: %s", estimatedArrival.Format("15:04")),
		)
		tx.Model(&lockedOrder).Update("status_history", lockedOrder.StatusHistory)

		// Update captain availability
		tx.Model(&captain).Updates(map[string]interface{}{
			"is_available":   false, // Captain busy with order
			"last_active_at": now,
		})

		order = &lockedOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Async notifications
	go s.notifyOrderAccepted(ctx, order)

	return order, nil
}

// IMPROVEMENT: CancelOrder with proper refund handling
func (s *Service) CancelOrder(ctx context.Context, orderID, userID uint, reason, cancelledBy string) (*models.Order, error) {
	lockKey := fmt.Sprintf("order:cancel:%d", orderID)

	locked, err := infra.Redis.SetNX(ctx, lockKey, userID, 5*time.Second).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return nil, errors.New("cancellation already in progress")
	}
	defer infra.Redis.Del(ctx, lockKey)

	var order *models.Order
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var lockedOrder models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&lockedOrder, orderID).Error; err != nil {
			return ErrOrderNotFound
		}

		// Authorization check
		isCustomer := lockedOrder.CustomerID == userID
		isCaptain := lockedOrder.CaptainID != nil && *lockedOrder.CaptainID == userID

		if !isCustomer && !isCaptain {
			return ErrUnauthorized
		}

		// Validate cancellation is allowed
		if !s.canCancelOrder(&lockedOrder, cancelledBy) {
			return errors.New("order cannot be cancelled at this stage")
		}

		// Calculate cancellation fee
		cancellationFee := lockedOrder.CalculateCancellationFee(cancelledBy)
		refundAmount := lockedOrder.CalculateRefundAmount()

		now := time.Now()
		updates := map[string]interface{}{
			"status":                     "CANCELLED",
			"cancelled_at":               now,
			"cancellation_reason":        reason,
			"cancelled_by":               cancelledBy,
			"cancellation_fee":           cancellationFee,
			"cancellation_refund_amount": refundAmount,
			"version":                    gorm.Expr("version + 1"),
		}

		result := tx.Model(&lockedOrder).
			Where("version = ?", lockedOrder.Version).
			Updates(updates)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrConcurrentUpdate
		}

		// Update status history
		lockedOrder.StatusHistory = s.addToStatusHistory(
			lockedOrder.StatusHistory,
			"CANCELLED",
			fmt.Sprintf("Cancelled by %s. Reason: %s", cancelledBy, reason),
		)
		tx.Model(&lockedOrder).Update("status_history", lockedOrder.StatusHistory)

		// Release captain if assigned
		if lockedOrder.CaptainID != nil {
			tx.Model(&models.User{}).
				Where("id = ?", *lockedOrder.CaptainID).
				Updates(map[string]interface{}{
					"is_available": true,
				})
		}

		// Process refund if payment was made
		if lockedOrder.PaymentStatus == "PAID" && refundAmount > 0 {
			// Create refund transaction
			refundTx := &models.Transaction{
				UserID:        lockedOrder.CustomerID,
				OrderID:       &orderID,
				TransactionID: generateTransactionID(),
				Type:          "REFUND",
				Amount:        refundAmount,
				Currency:      "INR",
				Status:        "PENDING",
				Description:   fmt.Sprintf("Refund for cancelled order %s", lockedOrder.OrderNumber),
			}

			if err := tx.Create(refundTx).Error; err != nil {
				return fmt.Errorf("failed to create refund transaction: %w", err)
			}

			// Queue refund processing
			go s.processRefund(ctx, refundTx)
		}

		order = &lockedOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Async notifications
	go s.notifyOrderCancelled(ctx, order)

	return order, nil
}

// IMPROVEMENT: Status transition validation
func (s *Service) canCancelOrder(order *models.Order, cancelledBy string) bool {
	// Define allowed cancellation states
	cancellableStates := map[string]bool{
		"PENDING":          true,
		"ACCEPTED":         true,
		"CAPTAIN_ARRIVING": true,
	}

	if !cancellableStates[order.Status] {
		return false
	}

	// Additional rules based on who's cancelling
	if cancelledBy == "CUSTOMER" {
		// Customer can't cancel if captain has picked up
		if order.Status == "PICKED_UP" || order.Status == "IN_TRANSIT" {
			return false
		}

		// Customer can't cancel if captain is very close (within 500m)
		if order.CaptainID != nil && order.Status == "CAPTAIN_ARRIVING" {
			var captain models.User
			if db.DB.First(&captain, *order.CaptainID).Error == nil {
				distance := calculateDistance(
					captain.CurrentLat, captain.CurrentLng,
					order.PickupLat, order.PickupLng,
				)
				if distance < 0.5 { // 500 meters
					return false
				}
			}
		}
	}

	return true
}

// StartPickup captain starts going to pickup location
func (s *Service) StartPickup(orderID, captainID uint) (*models.Order, error) {
	order, err := s.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	if order.CaptainID == nil || *order.CaptainID != captainID {
		return nil, ErrUnauthorized
	}

	if order.Status != "ACCEPTED" {
		return nil, ErrInvalidStatus
	}

	if err := db.DB.Model(order).Update("status", "CAPTAIN_ARRIVING").Error; err != nil {
		return nil, err
	}

	order.StatusHistory = s.addToStatusHistory(order.StatusHistory, "CAPTAIN_ARRIVING", "Captain on the way to pickup")
	db.DB.Model(order).Update("status_history", order.StatusHistory)

	return s.GetOrderByID(orderID)
}

// ConfirmPickup captain confirms pickup with OTP
func (s *Service) ConfirmPickup(orderID, captainID uint, otp string) (*models.Order, error) {
	order, err := s.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	if order.CaptainID == nil || *order.CaptainID != captainID {
		return nil, ErrUnauthorized
	}

	if order.PickupOTP != otp {
		return nil, errors.New("invalid OTP")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":       "PICKED_UP",
		"picked_up_at": now,
	}

	if err := db.DB.Model(order).Updates(updates).Error; err != nil {
		return nil, err
	}

	order.StatusHistory = s.addToStatusHistory(order.StatusHistory, "PICKED_UP", "Parcel picked up")
	db.DB.Model(order).Update("status_history", order.StatusHistory)

	return s.GetOrderByID(orderID)
}

// StartDelivery captain starts delivery to destination
func (s *Service) StartDelivery(orderID, captainID uint) (*models.Order, error) {
	order, err := s.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	if order.CaptainID == nil || *order.CaptainID != captainID {
		return nil, ErrUnauthorized
	}

	if order.Status != "PICKED_UP" {
		return nil, ErrInvalidStatus
	}

	if err := db.DB.Model(order).Update("status", "IN_TRANSIT").Error; err != nil {
		return nil, err
	}

	order.StatusHistory = s.addToStatusHistory(order.StatusHistory, "IN_TRANSIT", "On the way to delivery")
	db.DB.Model(order).Update("status_history", order.StatusHistory)

	return s.GetOrderByID(orderID)
}

// CompleteDelivery captain completes delivery with OTP
func (s *Service) CompleteDelivery(orderID, captainID uint, otp, proofPhotoURL string) (*models.Order, error) {
	order, err := s.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	if order.CaptainID == nil || *order.CaptainID != captainID {
		return nil, ErrUnauthorized
	}

	if order.DeliveryOTP != otp {
		return nil, errors.New("invalid OTP")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":          "DELIVERED",
		"delivered_at":    now,
		"is_otp_verified": true,
		"delivery_photo":  proofPhotoURL,
	}

	// Calculate actual duration
	if order.PickedUpAt != nil {
		actualDuration := int(now.Sub(*order.PickedUpAt).Minutes())
		updates["actual_duration"] = actualDuration
	}

	if err := db.DB.Model(order).Updates(updates).Error; err != nil {
		return nil, err
	}

	order.StatusHistory = s.addToStatusHistory(order.StatusHistory, "DELIVERED", "Parcel delivered successfully")
	db.DB.Model(order).Update("status_history", order.StatusHistory)

	// TODO: Update user statistics
	// TODO: Process payment
	// TODO: Update captain availability

	return s.GetOrderByID(orderID)
}

// Helper functions

func generateSecureOTP() string {
	// Use crypto/rand for secure OTP generation
	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	return otp
}

func generateOrderNumber() string {
	return fmt.Sprintf("ORD-%s-%s",
		time.Now().Format("20060102"),
		uuid.New().String()[:8])
}

func generateTransactionID() string {
	return fmt.Sprintf("TXN-%s-%s",
		time.Now().Format("20060102150405"),
		uuid.New().String()[:8])
}

func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371 // km
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

func estimateDuration(distanceKm float64) int {
	avgSpeed := 30.0                         // km/h
	return int((distanceKm / avgSpeed) * 60) // minutes
}

func (s *Service) createStatusHistory(status, note string) string {
	history := []map[string]interface{}{
		{
			"status":    status,
			"timestamp": time.Now(),
			"note":      note,
		},
	}
	historyJSON, _ := json.Marshal(history)
	return string(historyJSON)
}

func (s *Service) addToStatusHistory(currentHistory, status, note string) string {
	var history []map[string]interface{}
	if currentHistory != "" {
		json.Unmarshal([]byte(currentHistory), &history)
	}

	history = append(history, map[string]interface{}{
		"status":    status,
		"timestamp": time.Now(),
		"note":      note,
	})

	historyJSON, _ := json.Marshal(history)
	return string(historyJSON)
}

// Async notification helpers (implement these based on your notification service)
func (s *Service) notifyNearbyCaptains(ctx context.Context, order *models.Order) {
	// Implementation for notifying nearby captains
}

func (s *Service) sendOrderConfirmation(ctx context.Context, order *models.Order) {
	// Implementation for sending order confirmation
}

func (s *Service) notifyOrderAccepted(ctx context.Context, order *models.Order) {
	// Implementation for notifying customer that order was accepted
}

func (s *Service) notifyOrderCancelled(ctx context.Context, order *models.Order) {
	// Implementation for notifying about cancellation
}

func (s *Service) processRefund(ctx context.Context, transaction *models.Transaction) {
	// Implementation for processing refund via payment gateway
}
