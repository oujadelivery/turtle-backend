package valueobjects

import (
	"errors"
	"fmt"
	"math"
)

// Location represents a geographic coordinate
type Location struct {
	latitude  float64
	longitude float64
}

// NewLocation creates a new Location value object
func NewLocation(latitude, longitude float64) (*Location, error) {
	// Validate latitude (-90 to 90)
	if latitude < -90 || latitude > 90 {
		return nil, errors.New("latitude must be between -90 and 90")
	}

	// Validate longitude (-180 to 180)
	if longitude < -180 || longitude > 180 {
		return nil, errors.New("longitude must be between -180 and 180")
	}

	return &Location{
		latitude:  latitude,
		longitude: longitude,
	}, nil
}

// MustNewLocation creates Location or panics
func MustNewLocation(latitude, longitude float64) *Location {
	loc, err := NewLocation(latitude, longitude)
	if err != nil {
		panic(err)
	}
	return loc
}

// Latitude returns the latitude
func (l Location) Latitude() float64 {
	return l.latitude
}

// Longitude returns the longitude
func (l Location) Longitude() float64 {
	return l.longitude
}

// DistanceToKm calculates distance to another location in kilometers
// Uses Haversine formula
func (l Location) DistanceToKm(other Location) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := degreesToRadians(l.latitude)
	lat2Rad := degreesToRadians(other.latitude)
	deltaLat := degreesToRadians(other.latitude - l.latitude)
	deltaLon := degreesToRadians(other.longitude - l.longitude)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// DistanceToMeters calculates distance in meters
func (l Location) DistanceToMeters(other Location) float64 {
	return l.DistanceToKm(other) * 1000
}

// IsWithinRadiusKm checks if another location is within given radius in km
func (l Location) IsWithinRadiusKm(other Location, radiusKm float64) bool {
	return l.DistanceToKm(other) <= radiusKm
}

// Equals checks if two locations are equal (with small tolerance for floating point)
func (l Location) Equals(other Location) bool {
	const tolerance = 0.0001 // ~11 meters
	return math.Abs(l.latitude-other.latitude) < tolerance &&
		math.Abs(l.longitude-other.longitude) < tolerance
}

// String returns string representation
func (l Location) String() string {
	return fmt.Sprintf("(%.6f, %.6f)", l.latitude, l.longitude)
}

// degreesToRadians converts degrees to radians
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// BoundingBox returns a bounding box around the location with given radius in km
// Useful for database queries to find nearby items
type BoundingBox struct {
	MinLat float64
	MaxLat float64
	MinLng float64
	MaxLng float64
}

// GetBoundingBox returns a bounding box for efficient spatial queries
func (l Location) GetBoundingBox(radiusKm float64) BoundingBox {
	// Earth radius in km
	const earthRadiusKm = 6371.0

	// Angular distance in radians
	radDist := radiusKm / earthRadiusKm

	// Convert lat/lng to radians
	latRad := degreesToRadians(l.latitude)
	lngRad := degreesToRadians(l.longitude)

	// Calculate min/max latitudes
	minLat := latRad - radDist
	maxLat := latRad + radDist

	// Calculate min/max longitudes
	// Adjust for latitude (longitude circles get smaller near poles)
	deltaLng := math.Asin(math.Sin(radDist) / math.Cos(latRad))
	minLng := lngRad - deltaLng
	maxLng := lngRad + deltaLng

	// Convert back to degrees
	return BoundingBox{
		MinLat: radiansToDegrees(minLat),
		MaxLat: radiansToDegrees(maxLat),
		MinLng: radiansToDegrees(minLng),
		MaxLng: radiansToDegrees(maxLng),
	}
}

func radiansToDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}
