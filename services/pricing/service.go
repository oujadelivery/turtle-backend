package pricing

import (
	"fmt"
	"math"
	"time"

	"turtle/db"
	"turtle/models"

	"gorm.io/gorm"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

type PriceEstimate struct {
	BasePrice      float64
	DistancePrice  float64
	SurgePrice     float64
	TaxAmount      float64
	DiscountAmount float64
	TotalPrice     float64
	Currency       string
}

const (
	BasePriceINR       = 29.0 // Base pickup charge
	PricePerKmINR      = 12.0 // Price per km
	TaxPercent         = 5.0  // 5% GST
	PriorityMultiplier = 1.5  // 50% extra for priority
	MinimumPrice       = 49.0 // Minimum order value
)

// Parcel type multipliers
var parcelTypeMultipliers = map[string]float64{
	"DOCUMENT":    1.0,
	"PACKAGE":     1.2,
	"FOOD":        1.3,
	"FRAGILE":     1.5,
	"ELECTRONICS": 1.6,
	"CLOTHING":    1.1,
	"GROCERIES":   1.2,
	"MEDICINES":   1.4,
	"OTHER":       1.0,
}

// CalculatePrice calculates order price
func (s *Service) CalculatePrice(distanceKm float64, parcelType string, isPriority bool, couponCode string) PriceEstimate {
	// Base price
	basePrice := BasePriceINR

	// Distance price
	distancePrice := distanceKm * PricePerKmINR

	// Parcel type multiplier
	multiplier, exists := parcelTypeMultipliers[parcelType]
	if !exists {
		multiplier = 1.0
	}
	distancePrice *= multiplier

	// Priority multiplier
	if isPriority {
		distancePrice *= PriorityMultiplier
	}

	// Surge pricing (time-based)
	surgeMultiplier := s.calculateSurgeMultiplier()
	surgePrice := (basePrice + distancePrice) * (surgeMultiplier - 1)

	// Subtotal before tax and discount
	subtotal := basePrice + distancePrice + surgePrice

	// Apply minimum price
	if subtotal < MinimumPrice {
		subtotal = MinimumPrice
	}

	// Apply coupon discount
	discountAmount := 0.0
	if couponCode != "" {
		discountAmount = s.applyCoupon(couponCode, subtotal)
	}

	// Calculate tax on (subtotal - discount)
	taxableAmount := subtotal - discountAmount
	taxAmount := taxableAmount * TaxPercent / 100

	// Total price
	totalPrice := taxableAmount + taxAmount

	return PriceEstimate{
		BasePrice:      math.Round(basePrice*100) / 100,
		DistancePrice:  math.Round(distancePrice*100) / 100,
		SurgePrice:     math.Round(surgePrice*100) / 100,
		TaxAmount:      math.Round(taxAmount*100) / 100,
		DiscountAmount: math.Round(discountAmount*100) / 100,
		TotalPrice:     math.Round(totalPrice*100) / 100,
		Currency:       "INR",
	}
}

// calculateSurgeMultiplier calculates surge pricing based on time and demand
func (s *Service) calculateSurgeMultiplier() float64 {
	now := time.Now()
	hour := now.Hour()

	// Peak hours surge (8-10 AM, 6-9 PM)
	if (hour >= 8 && hour < 10) || (hour >= 18 && hour < 21) {
		return 1.3 // 30% surge
	}

	// Late night surge (11 PM - 6 AM)
	if hour >= 23 || hour < 6 {
		return 1.2 // 20% surge
	}

	// Weekend surge
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		if hour >= 10 && hour < 22 {
			return 1.15 // 15% surge on weekends
		}
	}

	// TODO: Add demand-based surge (count pending orders in area)

	return 1.0 // No surge
}

// applyCoupon applies coupon discount
func (s *Service) applyCoupon(code string, orderValue float64) float64 {
	var coupon models.Coupon
	if err := db.DB.Where("code = ? AND is_active = true", code).First(&coupon).Error; err != nil {
		return 0 // Coupon not found
	}

	// Validate coupon
	if !coupon.IsValid() {
		return 0
	}

	// Check minimum order value
	if orderValue < coupon.MinOrderValue {
		return 0
	}

	// Check usage limit
	if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
		return 0
	}

	// Calculate discount
	var discount float64
	switch coupon.Type {
	case "PERCENTAGE":
		discount = orderValue * coupon.Value / 100
	case "FIXED_AMOUNT":
		discount = coupon.Value
	case "FREE_DELIVERY":
		discount = BasePriceINR
	}

	// Apply max discount limit
	if coupon.MaxDiscountValue > 0 && discount > coupon.MaxDiscountValue {
		discount = coupon.MaxDiscountValue
	}

	return discount
}

// ValidateCoupon validates a coupon for an order
func (s *Service) ValidateCoupon(code string, orderValue float64, userID uint) (bool, string, float64) {
	var coupon models.Coupon
	if err := db.DB.Where("code = ? AND is_active = true", code).First(&coupon).Error; err != nil {
		return false, "Coupon not found", 0
	}

	if !coupon.IsValid() {
		return false, "Coupon has expired", 0
	}

	if orderValue < coupon.MinOrderValue {
		return false, fmt.Sprintf("Minimum order value is ₹%.2f", coupon.MinOrderValue), 0
	}

	if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
		return false, "Coupon usage limit exceeded", 0
	}

	// Check per-user limit
	if coupon.PerUserLimit > 0 {
		var usageCount int64
		db.DB.Model(&models.Order{}).Where("customer_id = ? AND coupon_code = ?", userID, code).Count(&usageCount)
		if usageCount >= int64(coupon.PerUserLimit) {
			return false, "You have already used this coupon", 0
		}
	}

	// Check first order only
	if coupon.FirstOrderOnly {
		var orderCount int64
		db.DB.Model(&models.Order{}).Where("customer_id = ?", userID).Count(&orderCount)
		if orderCount > 0 {
			return false, "This coupon is valid for first order only", 0
		}
	}

	// Calculate discount
	discount := s.applyCoupon(code, orderValue)
	// finalAmount := orderValue - discount

	return true, "Coupon applied successfully", discount
}

// IncrementCouponUsage increments coupon usage count
func (s *Service) IncrementCouponUsage(code string) error {
	return db.DB.Model(&models.Coupon{}).Where("code = ?", code).Update("usage_count", gorm.Expr("usage_count + 1")).Error
}
