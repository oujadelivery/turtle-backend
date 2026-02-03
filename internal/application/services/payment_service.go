package services

import (
	"context"
	"time"
	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
	"turtle/internal/domain/valueobjects"
)

// PaymentService handles payment operations
type PaymentService struct {
	transactionRepo domain.TransactionRepository
	userRepo        domain.UserRepository
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	transactionRepo domain.TransactionRepository,
	userRepo domain.UserRepository,
) *PaymentService {
	return &PaymentService{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
	}
}

// ProcessRefund processes refund for cancelled order
func (s *PaymentService) ProcessRefund(
	ctx context.Context,
	order *aggregates.Order,
	refundAmount *valueobjects.Money,
	reason string,
) error {
	if refundAmount == nil || refundAmount.IsZero() {
		return nil
	}

	paymentMethod := order.PaymentMethod()

	switch paymentMethod {
	case aggregates.PaymentMethodCash:
		// No refund needed for cash
		return nil

	case aggregates.PaymentMethodWallet:
		return s.refundToWallet(ctx, order, refundAmount, reason)

	case aggregates.PaymentMethodOnline:
		return s.refundToPaymentGateway(ctx, order, refundAmount, reason)

	default:
		return nil
	}
}

// refundToWallet refunds money to user's wallet
func (s *PaymentService) refundToWallet(
	ctx context.Context,
	order *aggregates.Order,
	amount *valueobjects.Money,
	reason string,
) error {
	// Get user
	user, err := s.userRepo.FindByID(ctx, order.BookedByUserID())
	if err != nil {
		return err
	}

	// Add to wallet
	if err := user.AddToWallet(amount); err != nil {
		return err
	}

	// Save user
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Record transaction
	transaction := &domain.Transaction{
		UserID:        order.BookedByUserID(),
		OrderID:       &order.ID(),
		Type:          "REFUND",
		Amount:        amount.AmountPaise(),
		Currency:      amount.Currency(),
		Status:        "COMPLETED",
		PaymentMethod: string(order.PaymentMethod()),
		Description:   reason,
		CreatedAt:     time.Now(),
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		// Log error but don't fail
	}

	return nil
}

// refundToPaymentGateway refunds via payment gateway
func (s *PaymentService) refundToPaymentGateway(
	ctx context.Context,
	order *aggregates.Order,
	amount *valueobjects.Money,
	reason string,
) error {
	// TODO: Integrate with payment gateway (Stripe/Razorpay)
	// For now, just record the refund request

	transaction := &domain.Transaction{
		UserID:          order.BookedByUserID(),
		OrderID:         &order.ID(),
		Type:            "REFUND",
		Amount:          amount.AmountPaise(),
		Currency:        amount.Currency(),
		Status:          "PENDING",
		PaymentMethod:   string(order.PaymentMethod()),
		PaymentIntentID: order.PaymentIntentID(),
		Description:     reason,
		CreatedAt:       time.Now(),
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return err
	}

	// TODO: Call payment gateway API
	// For now, mark as pending manual processing

	return nil
}
