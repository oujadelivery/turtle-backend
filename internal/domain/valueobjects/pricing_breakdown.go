package valueobjects

import (
	"errors"
)

// PricingBreakdown represents detailed pricing calculation
type PricingBreakdown struct {
	basePrice          *Money
	distancePrice      *Money
	weightPrice        *Money
	surgeMultiplier    float64
	surgeAmount        *Money
	waitingCharges     *Money
	insuranceFee       *Money
	platformCommission *Money
	taxAmount          *Money
	discountAmount     *Money

	// Final amounts
	subtotal       *Money
	total          *Money
	captainEarning *Money

	// Metadata
	distanceKm       float64
	estimatedMinutes int
}

// NewPricingBreakdown creates a new pricing breakdown
func NewPricingBreakdown(
	basePrice, distancePrice, weightPrice *Money,
	distanceKm float64,
	estimatedMinutes int,
) (*PricingBreakdown, error) {
	if basePrice == nil || distancePrice == nil {
		return nil, errors.New("base and distance prices are required")
	}

	if distanceKm < 0 {
		return nil, errors.New("distance cannot be negative")
	}

	if estimatedMinutes < 0 {
		return nil, errors.New("estimated minutes cannot be negative")
	}

	// Ensure all prices use same currency
	currency := basePrice.Currency()
	if distancePrice.Currency() != currency {
		return nil, errors.New("all prices must use the same currency")
	}
	if weightPrice != nil && weightPrice.Currency() != currency {
		return nil, errors.New("all prices must use the same currency")
	}

	// Calculate subtotal (before surge, waiting, etc.)
	subtotal, err := basePrice.Add(*distancePrice)
	if err != nil {
		return nil, err
	}

	if weightPrice != nil {
		subtotal, err = subtotal.Add(*weightPrice)
		if err != nil {
			return nil, err
		}
	}

	pb := &PricingBreakdown{
		basePrice:          basePrice,
		distancePrice:      distancePrice,
		weightPrice:        weightPrice,
		surgeMultiplier:    1.0, // No surge by default
		surgeAmount:        Zero(currency),
		waitingCharges:     Zero(currency),
		insuranceFee:       Zero(currency),
		platformCommission: Zero(currency),
		taxAmount:          Zero(currency),
		discountAmount:     Zero(currency),
		subtotal:           subtotal,
		total:              subtotal,
		captainEarning:     subtotal,
		distanceKm:         distanceKm,
		estimatedMinutes:   estimatedMinutes,
	}

	// Calculate platform commission and captain earning
	if err := pb.recalculateTotal(); err != nil {
		return nil, err
	}

	return pb, nil
}

// ApplySurge applies surge multiplier
func (pb *PricingBreakdown) ApplySurge(multiplier float64) error {
	if multiplier < 1.0 {
		return errors.New("surge multiplier must be >= 1.0")
	}
	if multiplier > 10.0 {
		return errors.New("surge multiplier cannot exceed 10.0")
	}

	pb.surgeMultiplier = multiplier

	// Calculate surge amount
	surgeFactor := multiplier - 1.0
	surgeAmount, err := pb.subtotal.Multiply(surgeFactor)
	if err != nil {
		return err
	}
	pb.surgeAmount = surgeAmount

	return pb.recalculateTotal()
}

// AddWaitingCharges adds waiting time charges
func (pb *PricingBreakdown) AddWaitingCharges(charges *Money) error {
	if charges == nil {
		return errors.New("waiting charges cannot be nil")
	}
	if charges.Currency() != pb.subtotal.Currency() {
		return errors.New("waiting charges must use the same currency")
	}
	if charges.IsNegative() {
		return errors.New("waiting charges cannot be negative")
	}

	pb.waitingCharges = charges
	return pb.recalculateTotal()
}

// SetInsurance sets insurance fee
func (pb *PricingBreakdown) SetInsurance(fee *Money) error {
	if fee == nil {
		return errors.New("insurance fee cannot be nil")
	}
	if fee.Currency() != pb.subtotal.Currency() {
		return errors.New("insurance fee must use the same currency")
	}
	if fee.IsNegative() {
		return errors.New("insurance fee cannot be negative")
	}

	pb.insuranceFee = fee
	return pb.recalculateTotal()
}

// ApplyDiscount applies discount amount
func (pb *PricingBreakdown) ApplyDiscount(discount *Money) error {
	if discount == nil {
		return errors.New("discount cannot be nil")
	}
	if discount.Currency() != pb.subtotal.Currency() {
		return errors.New("discount must use the same currency")
	}
	if discount.IsNegative() {
		return errors.New("discount cannot be negative")
	}

	pb.discountAmount = discount
	return pb.recalculateTotal()
}

// recalculateTotal updates total and captain earning
func (pb *PricingBreakdown) recalculateTotal() error {
	// Total = subtotal + surge + waiting + insurance - discount
	total := pb.subtotal

	components := []*Money{
		pb.surgeAmount,
		pb.waitingCharges,
		pb.insuranceFee,
	}

	var err error
	for _, component := range components {
		total, err = total.Add(*component)
		if err != nil {
			return err
		}
	}

	// Subtract discount
	if pb.discountAmount != nil && !pb.discountAmount.IsZero() {
		total, err = total.Subtract(*pb.discountAmount)
		if err != nil {
			return err
		}
	}

	// Ensure total is not negative
	if total.IsNegative() {
		total = Zero(total.Currency())
	}

	// Calculate platform commission (20%)
	commission, err := total.Multiply(0.20)
	if err != nil {
		return err
	}
	pb.platformCommission = commission

	// Calculate tax (if applicable - for now assume no tax or included in price)
	pb.taxAmount = Zero(total.Currency())

	pb.total = total

	// Captain earning = total - platform commission
	earning, err := total.Subtract(*commission)
	if err != nil {
		return err
	}
	pb.captainEarning = earning

	return nil
}

// Getters
func (pb PricingBreakdown) BasePrice() *Money          { return pb.basePrice }
func (pb PricingBreakdown) DistancePrice() *Money      { return pb.distancePrice }
func (pb PricingBreakdown) WeightPrice() *Money        { return pb.weightPrice }
func (pb PricingBreakdown) SurgeMultiplier() float64   { return pb.surgeMultiplier }
func (pb PricingBreakdown) SurgeAmount() *Money        { return pb.surgeAmount }
func (pb PricingBreakdown) WaitingCharges() *Money     { return pb.waitingCharges }
func (pb PricingBreakdown) InsuranceFee() *Money       { return pb.insuranceFee }
func (pb PricingBreakdown) PlatformCommission() *Money { return pb.platformCommission }
func (pb PricingBreakdown) TaxAmount() *Money          { return pb.taxAmount }
func (pb PricingBreakdown) DiscountAmount() *Money     { return pb.discountAmount }
func (pb PricingBreakdown) Subtotal() *Money           { return pb.subtotal }
func (pb PricingBreakdown) Total() *Money              { return pb.total }
func (pb PricingBreakdown) CaptainEarning() *Money     { return pb.captainEarning }
func (pb PricingBreakdown) DistanceKm() float64        { return pb.distanceKm }
func (pb PricingBreakdown) EstimatedMinutes() int      { return pb.estimatedMinutes }

// HasSurge checks if surge pricing is applied
func (pb PricingBreakdown) HasSurge() bool {
	return pb.surgeMultiplier > 1.0
}

// HasWaitingCharges checks if waiting charges are applied
func (pb PricingBreakdown) HasWaitingCharges() bool {
	return pb.waitingCharges != nil && !pb.waitingCharges.IsZero()
}

// HasInsurance checks if insurance is applied
func (pb PricingBreakdown) HasInsurance() bool {
	return pb.insuranceFee != nil && !pb.insuranceFee.IsZero()
}

// HasDiscount checks if discount is applied
func (pb PricingBreakdown) HasDiscount() bool {
	return pb.discountAmount != nil && !pb.discountAmount.IsZero()
}

// GetEffectiveRate returns price per km
func (pb PricingBreakdown) GetEffectiveRate() float64 {
	if pb.distanceKm == 0 {
		return 0
	}
	// Convert total to float for calculation
	totalPaise := float64(pb.total.Amount())
	return totalPaise / pb.distanceKm
}