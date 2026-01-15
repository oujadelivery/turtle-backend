package payment

import (
	"time"
	"turtle/db"
	"turtle/models"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GetWalletBalance retrieves user wallet balance
func (s *Service) GetWalletBalance(userID uint) (float64, error) {
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		return 0, err
	}
	return user.WalletBalance, nil
}

// AddMoneyToWallet adds money to wallet
func (s *Service) AddMoneyToWallet(userID uint, amount float64, paymentMethod string) (*models.Transaction, error) {
	var user models.User
	db.DB.First(&user, userID)
	now := time.Now()
	tx := &models.Transaction{
		UserID: userID,
		// TransactionID: generateTransactionID(),
		Type:          "WALLET_TOPUP",
		Amount:        amount,
		Currency:      "INR",
		BalanceBefore: user.WalletBalance,
		BalanceAfter:  user.WalletBalance + amount,
		PaymentMethod: paymentMethod,
		Status:        "SUCCESS",
		ProcessedAt:   &now,
	}

	db.DB.Create(tx)
	db.DB.Model(&user).Update("wallet_balance", user.WalletBalance+amount)

	return tx, nil
}

// CreateTransaction creates a transaction record
func (s *Service) CreateTransaction(userID uint, orderID *uint, txType string, amount float64) (*models.Transaction, error) {
	// Implementation similar to AddMoneyToWallet
	return nil, nil
}
