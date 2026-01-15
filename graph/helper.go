package graph

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"turtle/graph/model"
)

// getPaginationParams extracts page and pageSize from pagination input
func getPaginationParams(pagination *model.PaginationInput) (page, pageSize int) {
	page = 1
	pageSize = 20
	
	if pagination != nil {
		if pagination.Page != nil {
			page = *pagination.Page
		}
		if pagination.PageSize != nil {
			pageSize = *pagination.PageSize
		}
	}
	
	return page, pageSize
}

// Pointer to value conversions with defaults

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrToFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func ptrToInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func ptrToBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// Value to pointer conversions

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

// Optional pointer conversions (returns nil if empty/zero)

func optStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optIntPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func optFloatPtr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

// Enum Conversions - Convert GraphQL enums to strings

func enumToString[T ~string](e *T) string {
	if e == nil {
		return ""
	}
	return string(*e)
}

func stringToEnum[T ~string](s string) T {
	return T(s)
}

func ptrEnumToString[T ~string](e *T) string {
	if e == nil {
		return ""
	}
	return string(*e)
}

func stringToPtrEnum[T ~string](s string) *T {
	if s == "" {
		return nil
	}
	enum := T(s)
	return &enum
}

// Specific enum helpers for common types

func addressLabelToString(label *model.AddressLabel) string {
	if label == nil {
		return ""
	}
	return string(*label)
}

func stringToAddressLabel(s string) *model.AddressLabel {
	if s == "" {
		return nil
	}
	label := model.AddressLabel(s)
	return &label
}

// ID Generation helpers

func generateTransactionID() string {
	return fmt.Sprintf("TXN-%s-%06d", time.Now().Format("20060102150405"), rand.Intn(999999))
}

func generateOrderNumber() string {
	return fmt.Sprintf("ORD-%s-%06d", time.Now().Format("20060102"), rand.Intn(999999))
}

func generateOTP() string {
	return fmt.Sprintf("%06d", rand.Intn(999999))
}

func generateTicketNumber() string {
	now := time.Now()
	dateStr := now.Format("20060102")
	return fmt.Sprintf("TKT-%s-%06d", dateStr, rand.Intn(999999))
}

// Helper function
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
	// Rough estimate: 30 km/h average speed
	hours := distanceKm / 30.0
	return int(hours * 60) // convert to minutes
}