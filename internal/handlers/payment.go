// internal/handlers/payment.go
package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javajoker/imi-backend/internal/i18n"
	"github.com/javajoker/imi-backend/internal/services"
	"github.com/javajoker/imi-backend/internal/utils"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
}

func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// POST /payments/intent
func (h *PaymentHandler) CreatePaymentIntent(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.CreatePaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Create payment intent
	response, err := h.paymentService.CreatePaymentIntent(userID, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, response)
}

// POST /payments/confirm
func (h *PaymentHandler) ConfirmPayment(c *gin.Context) {
	lang := utils.GetLangFromContext(c)

	var req services.ConfirmPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Confirm payment
	if err := h.paymentService.ConfirmPayment(&req); err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyPaymentSuccess),
	})
}

// GET /payments/history
func (h *PaymentHandler) GetPaymentHistory(c *gin.Context) {
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	params := utils.GetPaginationParams(c)

	// Get payment history
	transactions, total, err := h.paymentService.GetPaymentHistory(userID, params)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(transactions, total, params)
	utils.PaginatedResponse(c, result)
}

// POST /payments/refund
func (h *PaymentHandler) ProcessRefund(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	// Check if user is admin (only admins can process refunds)
	userType, exists := utils.GetUserTypeFromContext(c)
	if !exists || userType != "admin" {
		utils.ForbiddenResponse(c, i18n.T(lang, i18n.KeyAdminAccessDenied))
		return
	}

	adminID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid admin ID", nil)
		return
	}

	var req services.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Process refund
	if err := h.paymentService.ProcessRefund(&req, &adminID); err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyPaymentRefunded),
	})
}

// GET /payments/balance
func (h *PaymentHandler) GetUserBalance(c *gin.Context) {
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	// Get user balance
	balance, err := h.paymentService.GetUserBalance(userID)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"balance": balance,
	})
}

// POST /payments/payout
func (h *PaymentHandler) RequestPayout(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.PayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Request payout
	if err := h.paymentService.RequestPayout(userID, &req); err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": "Payout request submitted successfully",
	})
}
