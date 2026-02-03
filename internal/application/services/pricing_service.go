package services

import (
	"context"
	"turtle/internal/domain/valueobjects"
)

// PricingService calculates order pricing
type PricingService struct {
	// Future: Add surge pricing, promo codes, etc.
}

// NewPricingService creates a new pricing service
func NewPricingService() *PricingService {
	return &PricingService{}
}

// CalculatePrice calculates order price based on distance and parcel details
func (s *PricingService) CalculatePrice(
	ctx context.Context,
	distanceKm float64,
	parcel *valueobjects.Parcel,
) (*valueobjects.PricingBreakdown, error) {
	// Base price: ₹50
	basePrice, _ := valueobjects.NewMoney(5000, "INR") // 5000 paise = ₹50

	// Distance price: ₹10/km
	pricePerKm, _ := valueobjects.NewMoney(1000, "INR") // 1000 paise = ₹10
	distancePrice, _ := pricePerKm.Multiply(distanceKm)

	// Weight price (if parcel)
	var weightPrice *valueobjects.Money
	if parcel != nil {
		// ₹5 per kg above 5kg
		if parcel.Weight() > 5.0 {
			extraWeight := parcel.Weight() - 5.0
			weightPricePerKg, _ := valueobjects.NewMoney(500, "INR")
			weightPrice, _ = weightPricePerKg.Multiply(extraWeight)
		}
	}

	// Estimate delivery time (15 min + 3 min per km)
	estimatedMinutes := 15 + int(distanceKm*3)

	pricing, err := valueobjects.NewPricingBreakdown(
		basePrice,
		distancePrice,
		weightPrice,
		distanceKm,
		estimatedMinutes,
	)
	if err != nil {
		return nil, err
	}

	// Apply insurance if needed
	if parcel != nil && parcel.RequiresInsurance() {
		insuranceFee := parcel.CalculateInsuranceFee()
		pricing.SetInsurance(insuranceFee)
	}

	// TODO: Check surge pricing
	// TODO: Apply promo codes

	return pricing, nil
}

// GetSurgeMultiplier calculates surge multiplier based on demand
func (s *PricingService) GetSurgeMultiplier(
	ctx context.Context,
	latitude, longitude float64,
) (float64, error) {
	// TODO: Implement surge pricing logic
	// - Check active orders in area
	// - Check available captains
	// - Calculate demand/supply ratio
	// - Return multiplier (1.0 = no surge, 2.0 = 2x surge)

	return 1.0, nil // No surge for now
}
