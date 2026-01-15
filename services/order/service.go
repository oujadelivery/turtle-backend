package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"turtle/db"
	"turtle/models"
	"turtle/services/pricing"

	"gorm.io/gorm"
)

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidStatus       = errors.New("invalid order status")
	ErrCaptainNotAvailable = errors.New("captain not available")
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
}

// CreateOrder creates a new delivery order
func (s *Service) CreateOrder(input CreateOrderInput) (*models.Order, error) {
	// Get addresses
	var pickupAddr, deliveryAddr models.Address
	if err := db.DB.First(&pickupAddr, input.PickupAddressID).Error; err != nil {
		return nil, errors.New("pickup address not found")
	}
	if err := db.DB.First(&deliveryAddr, input.DeliveryAddressID).Error; err != nil {
		return nil, errors.New("delivery address not found")
	}

	// Calculate distance and pricing
	distance := calculateDistance(pickupAddr.Latitude, pickupAddr.Longitude, deliveryAddr.Latitude, deliveryAddr.Longitude)
	duration := estimateDuration(distance)

	priceEstimate := s.pricingService.CalculatePrice(distance, input.ParcelType, input.IsPriority, input.CouponCode)

	// Generate order number
	orderNumber := generateOrderNumber()

	// Generate OTPs
	pickupOTP := generateOTP()
	deliveryOTP := generateOTP()

	// Create order
	order := &models.Order{
		OrderNumber:          orderNumber,
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
	}

	if len(input.ParcelImages) > 0 {
		imagesJSON, _ := json.Marshal(input.ParcelImages)
		order.ParcelImages = string(imagesJSON)
	}

	if input.IsInsured {
		order.InsuranceAmount = input.ParcelValue * 0.02 // 2% of parcel value
		order.TotalPrice += order.InsuranceAmount
	}

	// Calculate captain earnings (platform takes 20% commission)
	order.CalculateCaptainEarnings(20)

	// Add initial status to history
	order.StatusHistory = s.createStatusHistory("PENDING", "Order placed")

	if err := db.DB.Create(order).Error; err != nil {
		return nil, err
	}

	// TODO: Notify nearby captains
	go s.notifyNearbyCaptains(order)

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

// AcceptOrder captain accepts an order
func (s *Service) AcceptOrder(orderID, captainID uint) (*models.Order, error) {
	order, err := s.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != "PENDING" {
		return nil, errors.New("order is not available")
	}

	if order.CaptainID != nil {
		return nil, errors.New("order already accepted by another captain")
	}

	// Verify captain is available
	var captain models.User
	if err := db.DB.First(&captain, captainID).Error; err != nil {
		return nil, err
	}

	if !captain.CanTakeOrders() {
		return nil, ErrCaptainNotAvailable
	}

	now := time.Now()
	updates := map[string]interface{}{
		"captain_id":  captainID,
		"status":      "ACCEPTED",
		"accepted_at": now,
	}

	if err := db.DB.Model(order).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Update status history
	order.StatusHistory = s.addToStatusHistory(order.StatusHistory, "ACCEPTED", "Captain accepted order")
	db.DB.Model(order).Update("status_history", order.StatusHistory)

	// TODO: Notify customer

	return s.GetOrderByID(orderID)
}

// CancelOrder cancels an order
func (s *Service) CancelOrder(orderID, userID uint, reason, cancelledBy string) (*models.Order, error) {
	order, err := s.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if order.CustomerID != userID && (order.CaptainID == nil || *order.CaptainID != userID) {
		return nil, ErrUnauthorized
	}

	if !order.CanCancel() {
		return nil, errors.New("order cannot be cancelled at this stage")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":              "CANCELLED",
		"cancelled_at":        now,
		"cancellation_reason": reason,
		"cancelled_by":        cancelledBy,
	}

	if err := db.DB.Model(order).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Update status history
	order.StatusHistory = s.addToStatusHistory(order.StatusHistory, "CANCELLED", reason)
	db.DB.Model(order).Update("status_history", order.StatusHistory)

	// TODO: Process refund if payment was made
	// TODO: Update captain availability

	return s.GetOrderByID(orderID)
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

func generateOrderNumber() string {
	return fmt.Sprintf("ORD-%s-%06d", time.Now().Format("20060102"), rand.Intn(999999))
}

func generateOTP() string {
	return fmt.Sprintf("%06d", rand.Intn(999999))
}

func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Haversine formula
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
	// Rough estimate: 30 km/h average speed
	hours := distanceKm / 30.0
	return int(hours * 60) // convert to minutes
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

func (s *Service) notifyNearbyCaptains(order *models.Order) {
	// TODO: Implement push notification to nearby captains
	fmt.Printf("Notifying captains near order %s\n", order.OrderNumber)
}
