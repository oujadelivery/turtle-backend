package rating

import (
	"turtle/db"
	"turtle/models"
)

type Service struct{}

func NewService() *Service {
    return &Service{}
}

func (s *Service) CreateRating(orderID, userID, ratedUserID uint, rating float64, comment string) (*models.Rating, error) {
    r := &models.Rating{
        OrderID:     orderID,
        UserID:      userID,
        RatedUserID: ratedUserID,
        Rating:      rating,
        Comment:     comment,
    }

    if err := db.DB.Create(r).Error; err != nil {
        return nil, err
    }

    // Update user's average rating
    go s.updateUserRating(ratedUserID)

    return r, nil
}

func (s *Service) updateUserRating(userID uint) {
    var avgRating float64
    var count int64
    db.DB.Model(&models.Rating{}).Where("rated_user_id = ?", userID).
        Select("AVG(rating) as avg_rating, COUNT(*) as count").
        Row().Scan(&avgRating, &count)

    db.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
        "rating":        avgRating,
        "total_ratings": count,
    })
}