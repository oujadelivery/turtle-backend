package support

import (
	"errors"
	"fmt"
	"time"
	"turtle/db"
	"turtle/models"

	"gorm.io/gorm"
)

var (
	ErrTicketNotFound = errors.New("ticket not found")
	ErrUnauthorized   = errors.New("unauthorized")
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

type CreateTicketInput struct {
	UserID      uint
	OrderID     *uint
	Subject     string
	Description string
	Category    string
	Priority    string
	Attachments []string
}

// CreateTicket creates a new support ticket
func (s *Service) CreateTicket(input CreateTicketInput) (*models.SupportTicket, error) {
	// Generate ticket number
	ticketNumber := s.generateTicketNumber()

	// Convert attachments to JSON
	attachmentsJSON := ""
	if len(input.Attachments) > 0 {
		// Simple JSON array string
		attachmentsJSON = `["` + input.Attachments[0] + `"]`
	}

	ticket := &models.SupportTicket{
		UserID:       input.UserID,
		OrderID:      input.OrderID,
		TicketNumber: ticketNumber,
		Subject:      input.Subject,
		Description:  input.Description,
		Category:     input.Category,
		Status:       "OPEN",
		Priority:     input.Priority,
		Attachments:  attachmentsJSON,
	}

	if err := db.DB.Create(ticket).Error; err != nil {
		return nil, err
	}

	return s.GetTicketByID(ticket.ID)
}

// GetTicketByID retrieves ticket by ID
func (s *Service) GetTicketByID(ticketID uint) (*models.SupportTicket, error) {
	var ticket models.SupportTicket
	if err := db.DB.Preload("User").Preload("Order").Preload("Admin").First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// GetTicketByNumber retrieves ticket by ticket number
func (s *Service) GetTicketByNumber(ticketNumber string) (*models.SupportTicket, error) {
	var ticket models.SupportTicket
	if err := db.DB.Preload("User").Preload("Order").Preload("Admin").
		Where("ticket_number = ?", ticketNumber).First(&ticket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// GetUserTickets retrieves tickets for a user
func (s *Service) GetUserTickets(userID uint, status *string, page, pageSize int) ([]*models.SupportTicket, int64, error) {
	var tickets []*models.SupportTicket
	var total int64

	query := db.DB.Model(&models.SupportTicket{}).Where("user_id = ?", userID)

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Preload("Order").Order("created_at DESC").
		Offset(offset).Limit(pageSize).Find(&tickets).Error; err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

// GetAllTickets retrieves all tickets (admin)
func (s *Service) GetAllTickets(status, priority *string, page, pageSize int) ([]*models.SupportTicket, int64, error) {
	var tickets []*models.SupportTicket
	var total int64

	query := db.DB.Model(&models.SupportTicket{})

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	if priority != nil && *priority != "" {
		query = query.Where("priority = ?", *priority)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Preload("User").Preload("Order").Preload("Admin").
		Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tickets).Error; err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

// UpdateTicketStatus updates ticket status
func (s *Service) UpdateTicketStatus(ticketID uint, status, message *string) (*models.SupportTicket, error) {
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if status != nil {
		updates["status"] = *status

		if *status == "RESOLVED" {
			now := time.Now()
			updates["resolved_at"] = now
		}
	}

	if len(updates) > 0 {
		if err := db.DB.Model(ticket).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetTicketByID(ticketID)
}

// UpdateTicketPriority updates ticket priority
func (s *Service) UpdateTicketPriority(ticketID uint, priority string) (*models.SupportTicket, error) {
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	if err := db.DB.Model(ticket).Update("priority", priority).Error; err != nil {
		return nil, err
	}

	return s.GetTicketByID(ticketID)
}

// AssignTicket assigns ticket to admin
func (s *Service) AssignTicket(ticketID, adminID uint) (*models.SupportTicket, error) {
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"assigned_to_admin_id": adminID,
		"assigned_at":          now,
		"status":               "IN_PROGRESS",
	}

	if err := db.DB.Model(ticket).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetTicketByID(ticketID)
}

// CloseTicket closes a ticket
func (s *Service) CloseTicket(ticketID, userID uint) (*models.SupportTicket, error) {
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if ticket.UserID != userID {
		return nil, ErrUnauthorized
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      "CLOSED",
		"resolved_at": now,
	}

	if err := db.DB.Model(ticket).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetTicketByID(ticketID)
}

// AddTicketMessage adds a message/reply to ticket (placeholder)
func (s *Service) AddTicketMessage(ticketID uint, message string, attachments []string) (*models.SupportTicket, error) {
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	// TODO: Implement ticket messages as separate table
	// For now, just return the ticket

	return ticket, nil
}

// generateTicketNumber generates a unique ticket number
func (s *Service) generateTicketNumber() string {
	// Format: TKT-YYYYMMDD-NNNNNN
	now := time.Now()
	dateStr := now.Format("20060102")

	var count int64
	db.DB.Model(&models.SupportTicket{}).Where("DATE(created_at) = ?", now.Format("2006-01-02")).Count(&count)

	return fmt.Sprintf("TKT-%s-%06d", dateStr, count+1)
}
