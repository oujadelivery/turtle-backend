package location

import (
	"time"
	"turtle/db"
	"turtle/models"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) UpdateCurrentLocation(userID uint, lat, lng float64) (*models.CurrentLocation, error) {
	var loc models.CurrentLocation
	if err := db.DB.Where("user_id = ?", userID).First(&loc).Error; err != nil {
		// Create new
		loc = models.CurrentLocation{
			UserID:    userID,
			Latitude:  lat,
			Longitude: lng,
			Source:    "GPS",
		}
		db.DB.Create(&loc)
	} else {
		// Update
		db.DB.Model(&loc).Updates(map[string]interface{}{
			"latitude":  lat,
			"longitude": lng,
			"source":    "GPS",
		})
	}
	return &loc, nil
}

func (s *Service) TrackOrderLocation(orderID, captainID uint, lat, lng, speed, bearing float64) (*models.LocationTracking, error) {
	tracking := &models.LocationTracking{
		OrderID:    orderID,
		CaptainID:  captainID,
		Latitude:   lat,
		Longitude:  lng,
		Speed:      speed,
		Bearing:    bearing,
		RecordedAt: time.Now(),
	}
	db.DB.Create(tracking)
	return tracking, nil
}
