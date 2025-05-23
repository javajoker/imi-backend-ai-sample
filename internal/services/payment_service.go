// internal/services/payment_service.go
package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/refund"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/config"
	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/utils"
)

type PaymentService struct {
	db     *gorm.DB
	config *config.Config
}

type CreatePaymentIntentRequest struct {
	Amount        float64                `json:"amount" validate:"required,min=0.01"`
	Currency      string                 `json:"currency,omitempty"`
	PaymentMethod string                 `json:"payment_method" validate:"required"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentIntentResponse struct {
	ClientSecret string `json:"client_secret"`
	PaymentID    string `json:"payment_id"`
	Status       string `json:"status"`
}

type ConfirmPaymentRequest struct {
	PaymentIntentID string    `json:"payment_intent_id" validate:"required"`
	TransactionID   uuid.UUID `json:"transaction_id" validate:"required"`
}

type RefundRequest struct {
	TransactionID uuid.UUID `json:"transaction_id" validate:"required"`
	Amount        float64   `json:"amount,omitempty"`
	Reason        string    `json:"reason" validate:"required"`
}

type PayoutRequest struct {
	Amount      float64                `json:"amount" validate:"required,min=10"`
	Method      string                 `json:"method" validate:"required"`
	AccountInfo map[string]interface{} `json:"account_info,omitempty"`
}

func NewPaymentService(db *gorm.DB, config *config.Config) *PaymentService {
	// Initialize Stripe
	stripe.Key = config.Payment.StripeSecretKey

	return &PaymentService{
		db:     db,
		config: config,
	}
}

func (s *PaymentService) CreatePaymentIntent(userID uuid.UUID, req *CreatePaymentIntentRequest) (*PaymentIntentResponse, error) {
	// Set default currency
	currency := req.Currency
	if currency == "" {
		currency = "usd"
	}

	// Convert amount to cents for Stripe
	amountInCents := int64(req.Amount * 100)

	// Prepare metadata
	metadata := make(map[string]string)
	metadata["user_id"] = userID.String()
	for k, v := range req.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	// Create Stripe PaymentIntent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountInCents),
		Currency: stripe.String(currency),
	}

	// Add metadata
	for k, v := range metadata {
		params.AddMetadata(k, v)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	return &PaymentIntentResponse{
		ClientSecret: pi.ClientSecret,
		PaymentID:    pi.ID,
		Status:       string(pi.Status),
	}, nil
}

func (s *PaymentService) ConfirmPayment(req *ConfirmPaymentRequest) error {
	// Get payment intent from Stripe
	pi, err := paymentintent.Get(req.PaymentIntentID, nil)
	if err != nil {
		return fmt.Errorf("failed to get payment intent: %w", err)
	}

	// Find transaction
	var transaction models.Transaction
	if err := s.db.First(&transaction, req.TransactionID).Error; err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	// Update transaction based on payment status
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		now := time.Now()
		transaction.Status = models.TransactionStatusCompleted
		transaction.ProcessedAt = &now
		transaction.PaymentReference = pi.ID

		// Distribute revenue
		if err := s.distributeRevenue(&transaction); err != nil {
			// Log error but don't fail the payment confirmation
			fmt.Printf("Revenue distribution failed: %v\n", err)
		}

	case stripe.PaymentIntentStatusRequiresAction, stripe.PaymentIntentStatusRequiresConfirmation:
		transaction.Status = models.TransactionStatusPending

	default:
		transaction.Status = models.TransactionStatusFailed
	}

	if err := s.db.Save(&transaction).Error; err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

func (s *PaymentService) ProcessRefund(req *RefundRequest, adminID *uuid.UUID) error {
	// Find transaction
	var transaction models.Transaction
	if err := s.db.Preload("Buyer").Preload("Seller").
		First(&transaction, req.TransactionID).Error; err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	if transaction.Status != models.TransactionStatusCompleted {
		return errors.New("can only refund completed transactions")
	}

	// Calculate refund amount
	refundAmount := req.Amount
	if refundAmount <= 0 || refundAmount > transaction.Amount {
		refundAmount = transaction.Amount
	}

	// Process refund through Stripe if we have a payment reference
	if transaction.PaymentReference != "" {
		refundAmountCents := int64(refundAmount * 100)
		params := &stripe.RefundParams{
			PaymentIntent: stripe.String(transaction.PaymentReference),
			Amount:        stripe.Int64(refundAmountCents),
			Reason:        stripe.String("requested_by_customer"),
		}

		_, err := refund.New(params)
		if err != nil {
			return fmt.Errorf("failed to process refund: %w", err)
		}
	}

	// Update transaction
	now := time.Now()
	transaction.Status = models.TransactionStatusRefunded
	transaction.RefundedAt = &now
	transaction.RefundReason = req.Reason

	if err := s.db.Save(&transaction).Error; err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

func (s *PaymentService) GetPaymentHistory(userID uuid.UUID, params utils.PaginationParams) ([]models.Transaction, int64, error) {
	query := s.db.Model(&models.Transaction{}).
		Where("buyer_id = ? OR seller_id = ?", userID, userID).
		Preload("Product")

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "amount", "status"}
	query = utils.ApplySort(query, params, allowedSortFields)
	query = utils.ApplyPagination(query, params)

	// Execute query
	var transactions []models.Transaction
	if err := query.Find(&transactions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	return transactions, total, nil
}

func (s *PaymentService) GetUserBalance(userID uuid.UUID) (map[string]interface{}, error) {
	var totalEarnings, pendingPayouts, availableBalance float64

	// Calculate total earnings (as seller)
	s.db.Model(&models.Transaction{}).
		Where("seller_id = ? AND status = ?", userID, models.TransactionStatusCompleted).
		Select("COALESCE(SUM(amount - platform_fee), 0)").Scan(&totalEarnings)

	// Calculate pending payouts (placeholder - would be from a payouts table)
	pendingPayouts = 0

	// Available balance is total earnings minus pending payouts
	availableBalance = totalEarnings - pendingPayouts

	return map[string]interface{}{
		"total_earnings":    totalEarnings,
		"pending_payouts":   pendingPayouts,
		"available_balance": availableBalance,
		"currency":          "USD",
	}, nil
}

func (s *PaymentService) RequestPayout(userID uuid.UUID, req *PayoutRequest) error {
	// Verify user balance
	balance, err := s.GetUserBalance(userID)
	if err != nil {
		return fmt.Errorf("failed to get user balance: %w", err)
	}

	availableBalance := balance["available_balance"].(float64)
	if req.Amount > availableBalance {
		return errors.New("insufficient balance for payout")
	}

	if req.Amount < s.config.Payment.MinimumPayout {
		return fmt.Errorf("minimum payout amount is $%.2f", s.config.Payment.MinimumPayout)
	}

	// TODO: Create payout record (would be in a separate payouts table)
	// For now, we'll just log it
	fmt.Printf("Payout request: User %s, Amount: $%.2f, Method: %s\n",
		userID, req.Amount, req.Method)

	return nil
}

func (s *PaymentService) distributeRevenue(transaction *models.Transaction) error {
	// Parse revenue shares
	var revenueShares map[string]interface{}
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", transaction.RevenueShares)), &revenueShares); err != nil {
		return fmt.Errorf("failed to parse revenue shares: %w", err)
	}

	// In a real implementation, this would:
	// 1. Update user balances
	// 2. Create individual transaction records for each share
	// 3. Handle escrow for disputed transactions
	// 4. Send notifications to recipients

	fmt.Printf("Distributing revenue for transaction %s: %+v\n",
		transaction.ID, revenueShares)

	return nil
}
