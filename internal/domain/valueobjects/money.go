package valueobjects

import (
	"errors"
	"fmt"
)

// Money represents monetary value with currency
// Stored in smallest unit (paise for INR, cents for USD)
type Money struct {
	amount   int64  // Always in smallest unit
	currency string // ISO 4217 code
}

// NewMoney creates a new Money value object
func NewMoney(amount int64, currency string) (*Money, error) {
	if currency == "" {
		return nil, errors.New("currency cannot be empty")
	}

	// Validate currency (basic validation)
	if len(currency) != 3 {
		return nil, errors.New("currency must be 3-letter ISO code")
	}

	return &Money{
		amount:   amount,
		currency: currency,
	}, nil
}

// MustNewMoney creates Money or panics (use only with constants)
func MustNewMoney(amount int64, currency string) *Money {
	m, err := NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

// Amount returns the amount in smallest unit
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the currency code
func (m Money) Currency() string {
	return m.currency
}

// AmountInMajorUnit returns amount in major unit (e.g., rupees, dollars)
func (m Money) AmountInMajorUnit() float64 {
	return float64(m.amount) / 100.0
}

// Add adds two Money values (must be same currency)
func (m Money) Add(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, errors.New("cannot add money with different currencies")
	}
	return NewMoney(m.amount+other.amount, m.currency)
}

// Subtract subtracts two Money values (must be same currency)
func (m Money) Subtract(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, errors.New("cannot subtract money with different currencies")
	}
	return NewMoney(m.amount-other.amount, m.currency)
}

// Multiply multiplies money by a factor
func (m Money) Multiply(factor float64) (*Money, error) {
	newAmount := int64(float64(m.amount) * factor)
	return NewMoney(newAmount, m.currency)
}

// IsZero checks if amount is zero
func (m Money) IsZero() bool {
	return m.amount == 0
}

// IsNegative checks if amount is negative
func (m Money) IsNegative() bool {
	return m.amount < 0
}

// IsPositive checks if amount is positive
func (m Money) IsPositive() bool {
	return m.amount > 0
}

// Equals checks if two Money values are equal
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}

// GreaterThan checks if this money is greater than other
func (m Money) GreaterThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, errors.New("cannot compare money with different currencies")
	}
	return m.amount > other.amount, nil
}

// LessThan checks if this money is less than other
func (m Money) LessThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, errors.New("cannot compare money with different currencies")
	}
	return m.amount < other.amount, nil
}

// String returns string representation
func (m Money) String() string {
	return fmt.Sprintf("%s %.2f", m.currency, m.AmountInMajorUnit())
}

// Zero returns zero money in given currency
func Zero(currency string) *Money {
	return MustNewMoney(0, currency)
}

// FromMajorUnit creates Money from major unit (e.g., rupees, dollars)
// Example: FromMajorUnit(50.99, "INR") creates ₹50.99 (5099 paise)
func FromMajorUnit(amount float64, currency string) (*Money, error) {
	if currency == "" {
		return nil, errors.New("currency cannot be empty")
	}
	if len(currency) != 3 {
		return nil, errors.New("currency must be 3-letter ISO code")
	}

	// Convert to smallest unit (multiply by 100)
	smallestUnit := int64(amount * 100)

	return &Money{
		amount:   smallestUnit,
		currency: currency,
	}, nil
}

// MustFromMajorUnit creates Money from major unit or panics
func MustFromMajorUnit(amount float64, currency string) *Money {
	m, err := FromMajorUnit(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

// Divide divides money by a factor
func (m Money) Divide(divisor float64) (*Money, error) {
	if divisor == 0 {
		return nil, errors.New("cannot divide by zero")
	}
	newAmount := int64(float64(m.amount) / divisor)
	return NewMoney(newAmount, m.currency)
}

// GreaterThanOrEqual checks if this money is >= other
func (m Money) GreaterThanOrEqual(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, errors.New("cannot compare money with different currencies")
	}
	return m.amount >= other.amount, nil
}

// LessThanOrEqual checks if this money is <= other
func (m Money) LessThanOrEqual(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, errors.New("cannot compare money with different currencies")
	}
	return m.amount <= other.amount, nil
}

// Negate returns the negative of this money
func (m Money) Negate() *Money {
	return MustNewMoney(-m.amount, m.currency)
}

// Abs returns the absolute value of this money
func (m Money) Abs() *Money {
	if m.amount < 0 {
		return MustNewMoney(-m.amount, m.currency)
	}
	return MustNewMoney(m.amount, m.currency)
}

// Allocate splits money into N parts (useful for splitting bills)
// Returns exact amounts with remainder in first allocation
// Example: ₹100 split 3 ways = [₹33.34, ₹33.33, ₹33.33]
func (m Money) Allocate(n int) ([]*Money, error) {
	if n <= 0 {
		return nil, errors.New("number of allocations must be positive")
	}

	results := make([]*Money, n)
	baseAmount := m.amount / int64(n)
	remainder := m.amount % int64(n)

	for i := 0; i < n; i++ {
		amount := baseAmount
		if i < int(remainder) {
			amount++ // Distribute remainder to first few allocations
		}
		results[i] = MustNewMoney(amount, m.currency)
	}

	return results, nil
}

// AllocateByRatio splits money by ratios
// Example: ₹100 with ratios [3, 2] = [₹60, ₹40]
func (m Money) AllocateByRatio(ratios []int) ([]*Money, error) {
	if len(ratios) == 0 {
		return nil, errors.New("ratios cannot be empty")
	}

	// Calculate total ratio
	totalRatio := int64(0)
	for _, ratio := range ratios {
		if ratio < 0 {
			return nil, errors.New("ratios must be non-negative")
		}
		totalRatio += int64(ratio)
	}

	if totalRatio == 0 {
		return nil, errors.New("total ratio must be positive")
	}

	results := make([]*Money, len(ratios))
	allocated := int64(0)

	for i, ratio := range ratios {
		if i == len(ratios)-1 {
			// Last allocation gets remainder to avoid rounding issues
			results[i] = MustNewMoney(m.amount-allocated, m.currency)
		} else {
			amount := (m.amount * int64(ratio)) / totalRatio
			results[i] = MustNewMoney(amount, m.currency)
			allocated += amount
		}
	}

	return results, nil
}

// Format returns formatted string for display
// Format options: "symbol", "code", "simple"
func (m Money) Format(format string) string {
	switch format {
	case "symbol":
		symbol := getCurrencySymbol(m.currency)
		return fmt.Sprintf("%s%.2f", symbol, m.AmountInMajorUnit())
	case "code":
		return fmt.Sprintf("%s %.2f", m.currency, m.AmountInMajorUnit())
	case "simple":
		return fmt.Sprintf("%.2f", m.AmountInMajorUnit())
	default:
		return m.String()
	}
}

// getCurrencySymbol returns the symbol for common currencies
func getCurrencySymbol(currency string) string {
	symbols := map[string]string{
		"INR": "₹",
		"USD": "$",
		"EUR": "€",
		"GBP": "£",
		"JPY": "¥",
		"AUD": "A$",
		"CAD": "C$",
	}

	if symbol, ok := symbols[currency]; ok {
		return symbol
	}
	return currency + " "
}

// IsValidCurrency checks if currency code is valid ISO 4217
func IsValidCurrency(currency string) bool {
	// Common currencies - can be extended
	validCurrencies := map[string]bool{
		"INR": true, "USD": true, "EUR": true, "GBP": true,
		"JPY": true, "AUD": true, "CAD": true, "CHF": true,
		"CNY": true, "SEK": true, "NZD": true, "SGD": true,
	}

	return validCurrencies[currency]
}

// Min returns the minimum of two Money values
func Min(a, b Money) (*Money, error) {
	if a.currency != b.currency {
		return nil, errors.New("cannot compare money with different currencies")
	}
	if a.amount < b.amount {
		return &a, nil
	}
	return &b, nil
}

// Max returns the maximum of two Money values
func Max(a, b Money) (*Money, error) {
	if a.currency != b.currency {
		return nil, errors.New("cannot compare money with different currencies")
	}
	if a.amount > b.amount {
		return &a, nil
	}
	return &b, nil
}

// Sum returns the sum of multiple Money values (all must be same currency)
func Sum(amounts ...*Money) (*Money, error) {
	if len(amounts) == 0 {
		return nil, errors.New("cannot sum empty list")
	}

	result := amounts[0]
	for i := 1; i < len(amounts); i++ {
		var err error
		result, err = result.Add(*amounts[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
