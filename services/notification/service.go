package notification

import (
	"time"
	"turtle/db"
	"turtle/models"

	"gorm.io/gorm"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// CreateNotification creates a new notification
func (s *Service) CreateNotification(userID uint, title, message, notifType string, orderID, transactionID *uint, actionURL string) (*models.Notification, error) {
	notification := &models.Notification{
		UserID:        userID,
		Title:         title,
		Message:       message,
		Type:          notifType,
		OrderID:       orderID,
		TransactionID: transactionID,
		ActionURL:     actionURL,
		IsRead:        false,
		IsSent:        false,
	}

	if err := db.DB.Create(notification).Error; err != nil {
		return nil, err
	}

	// TODO: Send push notification via FCM/APNS
	go s.sendPushNotification(notification)

	return notification, nil
}

// GetUserNotifications retrieves user notifications with pagination
func (s *Service) GetUserNotifications(userID uint, isRead *bool, page, pageSize int) ([]*models.Notification, int64, error) {
	var notifications []*models.Notification
	var total int64

	query := db.DB.Model(&models.Notification{}).Where("user_id = ?", userID)

	if isRead != nil {
		query = query.Where("is_read = ?", *isRead)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(notificationID, userID uint) (*models.Notification, error) {
	var notification models.Notification
	if err := db.DB.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"is_read": true,
		"read_at": now,
	}

	if err := db.DB.Model(&notification).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &notification, nil
}

// MarkAllAsRead marks all user notifications as read
func (s *Service) MarkAllAsRead(userID uint) error {
	now := time.Now()
	return db.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// DeleteNotification deletes a notification
func (s *Service) DeleteNotification(notificationID, userID uint) error {
	result := db.DB.Where("id = ? AND user_id = ?", notificationID, userID).Delete(&models.Notification{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetUnreadCount returns count of unread notifications
func (s *Service) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := db.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count).Error
	return count, err
}

// UpdateDeviceToken updates user's device token for push notifications
func (s *Service) UpdateDeviceToken(userID uint, token, platform string) error {
	return db.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"device_token":    token,
			"device_platform": platform,
		}).Error
}

// sendPushNotification sends push notification (placeholder)
func (s *Service) sendPushNotification(notification *models.Notification) {
	// TODO: Implement FCM/APNS push notification
	// For now, just mark as sent
	db.DB.Model(notification).Updates(map[string]interface{}{
		"is_sent": true,
		"sent_at": time.Now(),
	})
}

// Helper methods for creating specific notification types

func (s *Service) NotifyOrderCreated(userID, orderID uint, orderNumber string) error {
	_, err := s.CreateNotification(
		userID,
		"Order Created",
		"Your order "+orderNumber+" has been placed successfully",
		"ORDER_UPDATE",
		&orderID,
		nil,
		"/orders/"+orderNumber,
	)
	return err
}

func (s *Service) NotifyOrderAccepted(userID, orderID uint, orderNumber string) error {
	_, err := s.CreateNotification(
		userID,
		"Order Accepted",
		"A captain has accepted your order "+orderNumber,
		"ORDER_UPDATE",
		&orderID,
		nil,
		"/orders/"+orderNumber,
	)
	return err
}

func (s *Service) NotifyOrderDelivered(userID, orderID uint, orderNumber string) error {
	_, err := s.CreateNotification(
		userID,
		"Order Delivered",
		"Your order "+orderNumber+" has been delivered",
		"ORDER_UPDATE",
		&orderID,
		nil,
		"/orders/"+orderNumber,
	)
	return err
}

func (s *Service) NotifyPaymentSuccess(userID, transactionID uint, amount float64) error {
	_, err := s.CreateNotification(
		userID,
		"Payment Successful",
		"Payment of ₹"+formatAmount(amount)+" completed successfully",
		"PAYMENT",
		nil,
		&transactionID,
		"/transactions",
	)
	return err
}

func formatAmount(amount float64) string {
	return sprintf("%.2f", amount)
}

func sprintf(format string, amount float64) string {
	// Simple float to string conversion
	return format
}
