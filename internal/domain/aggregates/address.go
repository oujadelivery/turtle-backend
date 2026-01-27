package aggregates

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"turtle/internal/domain/valueobjects"
)

// AddressLabel represents the label/type of address
type AddressLabel string

const (
	AddressLabelHome  AddressLabel = "HOME"
	AddressLabelWork  AddressLabel = "WORK"
	AddressLabelOther AddressLabel = "OTHER"
)

// TimeOfDay represents different time periods for analytics
type TimeOfDay string

const (
	TimeOfDayMorning   TimeOfDay = "MORNING"   // 6 AM - 12 PM
	TimeOfDayAfternoon TimeOfDay = "AFTERNOON" // 12 PM - 6 PM
	TimeOfDayEvening   TimeOfDay = "EVENING"   // 6 PM - 12 AM
	TimeOfDayNight     TimeOfDay = "NIGHT"     // 12 AM - 6 AM
)

// DayType represents weekday vs weekend
type DayType string

const (
	DayTypeWeekday DayType = "WEEKDAY" // Monday to Friday
	DayTypeWeekend DayType = "WEEKEND" // Saturday and Sunday
)

// Address entity with comprehensive analytics
type Address struct {
	// Identity
	id      string
	userID  string
	version int

	// Basic Information
	label        AddressLabel
	addressLine1 string
	addressLine2 string
	landmark     string
	city         string
	state        string
	country      string
	postalCode   string

	// Geolocation
	location *valueobjects.Location

	// Contact at this address
	contactName  string
	contactPhone string

	// Status
	isDefault  bool
	isActive   bool
	isVerified bool // Whether GPS coordinates are verified

	// Usage Statistics
	usageCount int
	lastUsedAt *time.Time

	// Time-based Analytics (for smart suggestions)
	morningUsageCount   int // 6 AM - 12 PM
	afternoonUsageCount int // 12 PM - 6 PM
	eveningUsageCount   int // 6 PM - 12 AM
	nightUsageCount     int // 12 AM - 6 AM

	// Day-based Analytics
	weekdayUsageCount int // Monday - Friday
	weekendUsageCount int // Saturday - Sunday

	// Month-based Analytics (optional, for seasonal patterns)
	monthlyUsageCount map[int]int // Month number (1-12) -> usage count

	// Timestamps
	createdAt time.Time
	updatedAt time.Time
	deletedAt *time.Time
}

// NewAddress creates a new address
func NewAddress(
	id, userID string,
	label AddressLabel,
	addressLine1, addressLine2, landmark string,
	city, state, country, postalCode string,
	location *valueobjects.Location,
	contactName, contactPhone string,
) (*Address, error) {
	// Validation
	if id == "" {
		return nil, errors.New("address ID is required")
	}
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	addressLine1 = strings.TrimSpace(addressLine1)
	if addressLine1 == "" {
		return nil, errors.New("address line 1 is required")
	}

	city = strings.TrimSpace(city)
	if city == "" {
		return nil, errors.New("city is required")
	}

	state = strings.TrimSpace(state)
	if state == "" {
		return nil, errors.New("state is required")
	}

	if location == nil {
		return nil, errors.New("location is required")
	}

	if country == "" {
		country = "India" // Default
	}

	now := time.Now()

	return &Address{
		id:                id,
		userID:            userID,
		version:           1,
		label:             label,
		addressLine1:      addressLine1,
		addressLine2:      addressLine2,
		landmark:          landmark,
		city:              city,
		state:             state,
		country:           country,
		postalCode:        postalCode,
		location:          location,
		contactName:       contactName,
		contactPhone:      contactPhone,
		isDefault:         false,
		isActive:          true,
		isVerified:        false,
		usageCount:        0,
		monthlyUsageCount: make(map[int]int),
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

// Getters
func (a *Address) ID() string                       { return a.id }
func (a *Address) UserID() string                   { return a.userID }
func (a *Address) Version() int                     { return a.version }
func (a *Address) Label() AddressLabel              { return a.label }
func (a *Address) AddressLine1() string             { return a.addressLine1 }
func (a *Address) AddressLine2() string             { return a.addressLine2 }
func (a *Address) Landmark() string                 { return a.landmark }
func (a *Address) City() string                     { return a.city }
func (a *Address) State() string                    { return a.state }
func (a *Address) Country() string                  { return a.country }
func (a *Address) PostalCode() string               { return a.postalCode }
func (a *Address) Location() *valueobjects.Location { return a.location }
func (a *Address) ContactName() string              { return a.contactName }
func (a *Address) ContactPhone() string             { return a.contactPhone }
func (a *Address) IsDefault() bool                  { return a.isDefault }
func (a *Address) IsActive() bool                   { return a.isActive }
func (a *Address) IsVerified() bool                 { return a.isVerified }
func (a *Address) UsageCount() int                  { return a.usageCount }
func (a *Address) LastUsedAt() *time.Time           { return a.lastUsedAt }
func (a *Address) MorningUsageCount() int           { return a.morningUsageCount }
func (a *Address) AfternoonUsageCount() int         { return a.afternoonUsageCount }
func (a *Address) EveningUsageCount() int           { return a.eveningUsageCount }
func (a *Address) NightUsageCount() int             { return a.nightUsageCount }
func (a *Address) WeekdayUsageCount() int           { return a.weekdayUsageCount }
func (a *Address) WeekendUsageCount() int           { return a.weekendUsageCount }
func (a *Address) MonthlyUsageCount() map[int]int   { return a.monthlyUsageCount }
func (a *Address) CreatedAt() time.Time             { return a.createdAt }
func (a *Address) UpdatedAt() time.Time             { return a.updatedAt }

// Business Methods

// Update updates address details
func (a *Address) Update(
	label AddressLabel,
	addressLine1, addressLine2, landmark string,
	city, state, postalCode string,
	location *valueobjects.Location,
	contactName, contactPhone string,
) error {
	addressLine1 = strings.TrimSpace(addressLine1)
	if addressLine1 == "" {
		return errors.New("address line 1 is required")
	}

	city = strings.TrimSpace(city)
	if city == "" {
		return errors.New("city is required")
	}

	state = strings.TrimSpace(state)
	if state == "" {
		return errors.New("state is required")
	}

	if location == nil {
		return errors.New("location is required")
	}

	a.label = label
	a.addressLine1 = addressLine1
	a.addressLine2 = addressLine2
	a.landmark = landmark
	a.city = city
	a.state = state
	a.postalCode = postalCode
	a.location = location
	a.contactName = contactName
	a.contactPhone = contactPhone
	a.updatedAt = time.Now()
	a.version++

	// If location changed, mark as unverified
	a.isVerified = false

	return nil
}

// SetAsDefault sets this address as the default
func (a *Address) SetAsDefault() {
	a.isDefault = true
	a.updatedAt = time.Now()
	a.version++
}

// UnsetDefault removes default status
func (a *Address) UnsetDefault() {
	a.isDefault = false
	a.updatedAt = time.Now()
}

// MarkAsVerified marks the address location as verified
func (a *Address) MarkAsVerified() {
	a.isVerified = true
	a.updatedAt = time.Now()
	a.version++
}

// Deactivate deactivates the address (soft delete)
func (a *Address) Deactivate() {
	a.isActive = false
	now := time.Now()
	a.deletedAt = &now
	a.updatedAt = now
	a.version++
}

// Reactivate reactivates a deactivated address
func (a *Address) Reactivate() {
	a.isActive = true
	a.deletedAt = nil
	a.updatedAt = time.Now()
	a.version++
}

// IncrementUsage increments usage statistics based on current time
func (a *Address) IncrementUsage() {
	now := time.Now()

	// Increment total usage
	a.usageCount++
	a.lastUsedAt = &now

	// Time-based increment
	hour := now.Hour()
	switch {
	case hour >= 6 && hour < 12:
		a.morningUsageCount++
	case hour >= 12 && hour < 18:
		a.afternoonUsageCount++
	case hour >= 18 && hour < 24:
		a.eveningUsageCount++
	default: // 0-6
		a.nightUsageCount++
	}

	// Day-based increment
	weekday := now.Weekday()
	if weekday >= time.Monday && weekday <= time.Friday {
		a.weekdayUsageCount++
	} else {
		a.weekendUsageCount++
	}

	// Month-based increment
	month := int(now.Month())
	a.monthlyUsageCount[month]++

	a.updatedAt = now
	// Don't increment version for analytics updates
}

// GetFullAddress returns formatted full address string
func (a *Address) GetFullAddress() string {
	parts := []string{a.addressLine1}

	if a.addressLine2 != "" {
		parts = append(parts, a.addressLine2)
	}
	if a.landmark != "" {
		parts = append(parts, a.landmark)
	}
	parts = append(parts, a.city)
	parts = append(parts, a.state)

	if a.postalCode != "" {
		parts = append(parts, a.postalCode)
	}

	return strings.Join(parts, ", ")
}

// GetShortAddress returns a shortened version for display
func (a *Address) GetShortAddress() string {
	if a.label != AddressLabelOther {
		return fmt.Sprintf("%s - %s", a.label, a.city)
	}
	return fmt.Sprintf("%s, %s", a.addressLine1, a.city)
}

// Analytics Methods

// GetPreferredTimeSlot returns when this address is most frequently used
func (a *Address) GetPreferredTimeSlot() TimeOfDay {
	counts := map[TimeOfDay]int{
		TimeOfDayMorning:   a.morningUsageCount,
		TimeOfDayAfternoon: a.afternoonUsageCount,
		TimeOfDayEvening:   a.eveningUsageCount,
		TimeOfDayNight:     a.nightUsageCount,
	}

	max := 0
	preferred := TimeOfDayMorning
	for slot, count := range counts {
		if count > max {
			max = count
			preferred = slot
		}
	}
	return preferred
}

// GetPreferredDayType returns whether this address is used more on weekdays or weekends
func (a *Address) GetPreferredDayType() DayType {
	if a.weekdayUsageCount >= a.weekendUsageCount {
		return DayTypeWeekday
	}
	return DayTypeWeekend
}

// GetMostUsedMonth returns the month with highest usage
func (a *Address) GetMostUsedMonth() int {
	if len(a.monthlyUsageCount) == 0 {
		return int(time.Now().Month())
	}

	maxMonth := 1
	maxCount := 0
	for month, count := range a.monthlyUsageCount {
		if count > maxCount {
			maxCount = count
			maxMonth = month
		}
	}
	return maxMonth
}

// IsLikelyHomeAddress returns true if usage pattern suggests this is a home address
func (a *Address) IsLikelyHomeAddress() bool {
	// Home addresses typically have:
	// 1. High evening/night usage
	// 2. Relatively balanced weekday/weekend usage
	// 3. High overall usage count

	totalTimeUsage := a.morningUsageCount + a.afternoonUsageCount +
		a.eveningUsageCount + a.nightUsageCount

	if totalTimeUsage == 0 {
		return false
	}

	eveningNightUsage := a.eveningUsageCount + a.nightUsageCount
	eveningNightPercentage := float64(eveningNightUsage) / float64(totalTimeUsage)

	// If >60% usage is in evening/night, likely home
	return eveningNightPercentage > 0.6 && a.usageCount >= 5
}

// IsLikelyWorkAddress returns true if usage pattern suggests this is a work address
func (a *Address) IsLikelyWorkAddress() bool {
	// Work addresses typically have:
	// 1. High morning/afternoon usage
	// 2. Much higher weekday than weekend usage
	// 3. Low evening/night usage

	totalTimeUsage := a.morningUsageCount + a.afternoonUsageCount +
		a.eveningUsageCount + a.nightUsageCount

	if totalTimeUsage == 0 {
		return false
	}

	morningAfternoonUsage := a.morningUsageCount + a.afternoonUsageCount
	morningAfternoonPercentage := float64(morningAfternoonUsage) / float64(totalTimeUsage)

	totalDayUsage := a.weekdayUsageCount + a.weekendUsageCount
	if totalDayUsage == 0 {
		return false
	}

	weekdayPercentage := float64(a.weekdayUsageCount) / float64(totalDayUsage)

	// If >70% usage is morning/afternoon AND >80% is weekdays, likely work
	return morningAfternoonPercentage > 0.7 &&
		weekdayPercentage > 0.8 &&
		a.usageCount >= 5
}

// ShouldSuggestAt returns true if this address should be suggested at the given time
func (a *Address) ShouldSuggestAt(t time.Time) bool {
	if !a.isActive {
		return false
	}

	// Must have some usage history
	if a.usageCount < 2 {
		return false
	}

	currentTimeSlot := getTimeSlot(t)
	currentDayType := getDayType(t)

	// Check if current time slot matches preferred pattern
	preferredTimeSlot := a.GetPreferredTimeSlot()
	preferredDayType := a.GetPreferredDayType()

	// If current conditions match historical pattern, suggest
	return currentTimeSlot == preferredTimeSlot || currentDayType == preferredDayType
}

// GetUsageScore returns a score (0-100) indicating how frequently this address is used
// considering recency and frequency
func (a *Address) GetUsageScore() float64 {
	if a.usageCount == 0 {
		return 0
	}

	// Frequency score (0-50 points)
	frequencyScore := float64(a.usageCount)
	if frequencyScore > 50 {
		frequencyScore = 50
	}

	// Recency score (0-50 points)
	recencyScore := 0.0
	if a.lastUsedAt != nil {
		daysSinceLastUse := time.Since(*a.lastUsedAt).Hours() / 24
		if daysSinceLastUse <= 1 {
			recencyScore = 50
		} else if daysSinceLastUse <= 7 {
			recencyScore = 40
		} else if daysSinceLastUse <= 30 {
			recencyScore = 30
		} else if daysSinceLastUse <= 90 {
			recencyScore = 20
		} else {
			recencyScore = 10
		}
	}

	return frequencyScore + recencyScore
}

// Helper functions

func getTimeSlot(t time.Time) TimeOfDay {
	hour := t.Hour()
	switch {
	case hour >= 6 && hour < 12:
		return TimeOfDayMorning
	case hour >= 12 && hour < 18:
		return TimeOfDayAfternoon
	case hour >= 18 && hour < 24:
		return TimeOfDayEvening
	default:
		return TimeOfDayNight
	}
}

func getDayType(t time.Time) DayType {
	weekday := t.Weekday()
	if weekday >= time.Monday && weekday <= time.Friday {
		return DayTypeWeekday
	}
	return DayTypeWeekend
}
