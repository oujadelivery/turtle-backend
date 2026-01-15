package chat

import (
	"errors"
	"time"
	"turtle/db"
	"turtle/graph/model"
	"turtle/models"

	"gorm.io/gorm"
)

var (
	ErrRoomNotFound    = errors.New("chat room not found")
	ErrMessageNotFound = errors.New("message not found")
	ErrUnauthorized    = errors.New("unauthorized")
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GetOrCreateChatRoom gets or creates a chat room for an order
func (s *Service) GetOrCreateChatRoom(orderID, customerID, captainID uint) (*models.ChatRoom, error) {
	var room models.ChatRoom

	// Try to find existing room
	err := db.DB.Where("order_id = ?", orderID).First(&room).Error
	if err == nil {
		return &room, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create new room
	room = models.ChatRoom{
		OrderID:    orderID,
		CustomerID: customerID,
		CaptainID:  captainID,
		IsActive:   true,
	}

	if err := db.DB.Create(&room).Error; err != nil {
		return nil, err
	}

	return &room, nil
}

// GetChatRoom retrieves chat room by ID
func (s *Service) GetChatRoom(roomID uint) (*models.ChatRoom, error) {
	var room models.ChatRoom
	if err := db.DB.Preload("Order").Preload("Customer").Preload("Captain").
		First(&room, roomID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return &room, nil
}

// GetChatRoomByOrder retrieves chat room by order ID
func (s *Service) GetChatRoomByOrder(orderID uint) (*models.ChatRoom, error) {
	var room models.ChatRoom
	if err := db.DB.Preload("Order").Preload("Customer").Preload("Captain").
		Where("order_id = ?", orderID).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return &room, nil
}

// SendMessage sends a chat message
func (s *Service) SendMessage(roomID, senderID, receiverID uint, message string, messageType *model.MessageType, mediaURL *string, lat, lng *float64) (*models.ChatMessage, error) {
	// Verify room exists and user has access
	room, err := s.GetChatRoom(roomID)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if room.CustomerID != senderID && room.CaptainID != senderID {
		return nil, ErrUnauthorized
	}

	//@@@@@@@@@@@@ changed messageType to string for now @@@@@@@@@@@@ //
	now := time.Now()
	chatMessage := &models.ChatMessage{
		RoomID:      roomID,
		SenderID:    senderID,
		ReceiverID:  receiverID,
		Message:     message,
		MessageType: "messageType",
		IsRead:      false,
	}

	if mediaURL != nil {
		chatMessage.MediaURL = *mediaURL
	}

	if lat != nil && lng != nil {
		chatMessage.Latitude = *lat
		chatMessage.Longitude = *lng
	}

	if err := db.DB.Create(chatMessage).Error; err != nil {
		return nil, err
	}

	// Update room's last message timestamp
	db.DB.Model(room).Update("last_message_at", now)

	return chatMessage, nil
}

// GetMessages retrieves messages for a chat room
func (s *Service) GetMessages(roomID, userID uint, page, pageSize int) ([]*models.ChatMessage, int64, error) {
	// Verify user has access to room
	room, err := s.GetChatRoom(roomID)
	if err != nil {
		return nil, 0, err
	}

	if room.CustomerID != userID && room.CaptainID != userID {
		return nil, 0, ErrUnauthorized
	}

	var messages []*models.ChatMessage
	var total int64

	query := db.DB.Model(&models.ChatMessage{}).Where("room_id = ?", roomID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Preload("Sender").Preload("Receiver").
		Order("created_at ASC").Offset(offset).Limit(pageSize).
		Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// MarkMessagesAsRead marks messages as read
func (s *Service) MarkMessagesAsRead(roomID, userID uint) error {
	now := time.Now()
	return db.DB.Model(&models.ChatMessage{}).
		Where("room_id = ? AND receiver_id = ? AND is_read = false", roomID, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// GetUnreadCount gets unread message count for user
func (s *Service) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := db.DB.Model(&models.ChatMessage{}).
		Where("receiver_id = ? AND is_read = false", userID).
		Count(&count).Error
	return count, err
}

// GetUserChatRooms gets all chat rooms for a user
func (s *Service) GetUserChatRooms(userID uint) ([]*models.ChatRoom, error) {
	var rooms []*models.ChatRoom
	if err := db.DB.Preload("Order").Preload("Customer").Preload("Captain").
		Where("customer_id = ? OR captain_id = ?", userID, userID).
		Where("is_active = true").
		Order("last_message_at DESC").
		Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

// DeactivateRoom deactivates a chat room
func (s *Service) DeactivateRoom(roomID uint) error {
	return db.DB.Model(&models.ChatRoom{}).
		Where("id = ?", roomID).
		Update("is_active", false).Error
}
