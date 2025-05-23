// internal/handlers/admin.go
package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javajoker/imi-backend/internal/i18n"
	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/services"
	"github.com/javajoker/imi-backend/internal/utils"
)

type AdminHandler struct {
	adminService *services.AdminService
}

func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// GET /admin/dashboard/stats
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	stats, err := h.adminService.GetDashboardStats()
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"stats": stats,
	})
}

// GET /admin/users
func (h *AdminHandler) GetUsers(c *gin.Context) {
	params := utils.GetPaginationParams(c)

	// Build filter parameters
	filter := services.AdminUserFilter{
		PaginationParams: params,
	}

	// Parse filters
	if userType := c.Query("user_type"); userType != "" {
		uType := models.UserType(userType)
		filter.UserType = &uType
	}

	if status := c.Query("status"); status != "" {
		uStatus := models.UserStatus(status)
		filter.Status = &uStatus
	}

	if verificationLevel := c.Query("verification_level"); verificationLevel != "" {
		vLevel := models.VerificationLevel(verificationLevel)
		filter.VerificationLevel = &vLevel
	}

	if createdAfter := c.Query("created_after"); createdAfter != "" {
		if t, err := time.Parse("2006-01-02", createdAfter); err == nil {
			filter.CreatedAfter = &t
		}
	}

	if createdBefore := c.Query("created_before"); createdBefore != "" {
		if t, err := time.Parse("2006-01-02", createdBefore); err == nil {
			filter.CreatedBefore = &t
		}
	}

	// Get users
	users, total, err := h.adminService.GetUsers(filter)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(users, total, params)
	utils.PaginatedResponse(c, result)
}

// PUT /admin/users/:id/status
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	adminIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid admin ID", nil)
		return
	}

	var req struct {
		Status models.UserStatus `json:"status" validate:"required"`
		Reason string            `json:"reason,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Update user status
	if err := h.adminService.UpdateUserStatus(userID, req.Status, adminID, req.Reason); err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyUserNotFound)
			return
		}
		if strings.Contains(err.Error(), "cannot modify") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	var message string
	switch req.Status {
	case models.UserStatusSuspended:
		message = i18n.T(lang, i18n.KeyAdminUserSuspended)
	case models.UserStatusActive:
		message = i18n.T(lang, i18n.KeyAdminUserUnsuspended)
	default:
		message = i18n.T(lang, i18n.KeyAdminActionSuccess)
	}

	utils.SuccessResponse(c, gin.H{
		"message": message,
	})
}

// PUT /admin/users/:id/verify
func (h *AdminHandler) UpdateUserVerification(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	adminIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid admin ID", nil)
		return
	}

	var req struct {
		VerificationLevel models.VerificationLevel `json:"verification_level" validate:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Update verification level
	if err := h.adminService.UpdateUserVerificationLevel(userID, req.VerificationLevel, adminID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyUserNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyAdminActionSuccess),
	})
}

// GET /admin/ip-assets/pending
func (h *AdminHandler) GetPendingIPAssets(c *gin.Context) {
	params := utils.GetPaginationParams(c)

	// Build filter parameters
	filter := services.AdminIPFilter{
		PaginationParams: params,
	}

	// Parse filters
	if verificationStatus := c.Query("verification_status"); verificationStatus != "" {
		vStatus := models.VerificationStatus(verificationStatus)
		filter.VerificationStatus = &vStatus
	} else {
		// Default to pending
		pending := models.VerificationStatusPending
		filter.VerificationStatus = &pending
	}

	if creatorIDStr := c.Query("creator_id"); creatorIDStr != "" {
		if creatorID, err := uuid.Parse(creatorIDStr); err == nil {
			filter.CreatorID = &creatorID
		}
	}

	if status := c.Query("status"); status != "" {
		pStatus := models.ProductStatus(status)
		filter.Status = &pStatus
	}

	if createdAfter := c.Query("created_after"); createdAfter != "" {
		if t, err := time.Parse("2006-01-02", createdAfter); err == nil {
			filter.CreatedAfter = &t
		}
	}

	if createdBefore := c.Query("created_before"); createdBefore != "" {
		if t, err := time.Parse("2006-01-02", createdBefore); err == nil {
			filter.CreatedBefore = &t
		}
	}

	// Get pending IP assets
	ipAssets, total, err := h.adminService.GetPendingIPAssets(filter)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(ipAssets, total, params)
	utils.PaginatedResponse(c, result)
}

// PUT /admin/ip-assets/:id/approve
func (h *AdminHandler) ApproveIPAsset(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	ipAssetID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid IP asset ID", nil)
		return
	}

	adminIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid admin ID", nil)
		return
	}

	var req struct {
		Message string `json:"message,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Message is optional, so we can ignore binding errors
		req.Message = ""
	}

	// Approve IP asset
	if err := h.adminService.ApproveIPAsset(ipAssetID, adminID, req.Message); err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyIPAssetNotFound)
			return
		}
		if strings.Contains(err.Error(), "not pending") {
			utils.BadRequestResponse(c, err.Error(), nil)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyIPAssetApproved),
	})
}

// PUT /admin/ip-assets/:id/reject
func (h *AdminHandler) RejectIPAsset(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	ipAssetID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid IP asset ID", nil)
		return
	}

	adminIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid admin ID", nil)
		return
	}

	var req struct {
		Reason  string `json:"reason" validate:"required"`
		Message string `json:"message,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Reject IP asset
	if err := h.adminService.RejectIPAsset(ipAssetID, adminID, req.Reason, req.Message); err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyIPAssetNotFound)
			return
		}
		if strings.Contains(err.Error(), "not pending") {
			utils.BadRequestResponse(c, err.Error(), nil)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyIPAssetRejected),
	})
}

// GET /admin/licenses
func (h *AdminHandler) GetLicenseApplications(c *gin.Context) {
	params := utils.GetPaginationParams(c)

	// Build filter parameters
	filter := services.AdminLicenseFilter{
		PaginationParams: params,
	}

	// Parse filters
	if applicantIDStr := c.Query("applicant_id"); applicantIDStr != "" {
		if applicantID, err := uuid.Parse(applicantIDStr); err == nil {
			filter.ApplicantID = &applicantID
		}
	}

	if ipAssetIDStr := c.Query("ip_asset_id"); ipAssetIDStr != "" {
		if ipAssetID, err := uuid.Parse(ipAssetIDStr); err == nil {
			filter.IPAssetID = &ipAssetID
		}
	}

	if status := c.Query("status"); status != "" {
		appStatus := models.ApplicationStatus(status)
		filter.Status = &appStatus
	}

	if licenseType := c.Query("license_type"); licenseType != "" {
		lType := models.LicenseType(licenseType)
		filter.LicenseType = &lType
	}

	if createdAfter := c.Query("created_after"); createdAfter != "" {
		if t, err := time.Parse("2006-01-02", createdAfter); err == nil {
			filter.CreatedAfter = &t
		}
	}

	if createdBefore := c.Query("created_before"); createdBefore != "" {
		if t, err := time.Parse("2006-01-02", createdBefore); err == nil {
			filter.CreatedBefore = &t
		}
	}

	// Get license applications
	applications, total, err := h.adminService.GetLicenseApplications(filter)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(applications, total, params)
	utils.PaginatedResponse(c, result)
}

// GET /admin/transactions
func (h *AdminHandler) GetTransactions(c *gin.Context) {
	params := utils.GetPaginationParams(c)

	// Build filter parameters
	filter := services.AdminTransactionFilter{
		PaginationParams: params,
	}

	// Parse filters
	if transactionType := c.Query("transaction_type"); transactionType != "" {
		tType := models.TransactionType(transactionType)
		filter.TransactionType = &tType
	}

	if status := c.Query("status"); status != "" {
		tStatus := models.TransactionStatus(status)
		filter.Status = &tStatus
	}

	if buyerIDStr := c.Query("buyer_id"); buyerIDStr != "" {
		if buyerID, err := uuid.Parse(buyerIDStr); err == nil {
			filter.BuyerID = &buyerID
		}
	}

	if sellerIDStr := c.Query("seller_id"); sellerIDStr != "" {
		if sellerID, err := uuid.Parse(sellerIDStr); err == nil {
			filter.SellerID = &sellerID
		}
	}

	if amountMinStr := c.Query("amount_min"); amountMinStr != "" {
		if amountMin, err := strconv.ParseFloat(amountMinStr, 64); err == nil {
			filter.AmountMin = &amountMin
		}
	}

	if amountMaxStr := c.Query("amount_max"); amountMaxStr != "" {
		if amountMax, err := strconv.ParseFloat(amountMaxStr, 64); err == nil {
			filter.AmountMax = &amountMax
		}
	}

	if createdAfter := c.Query("created_after"); createdAfter != "" {
		if t, err := time.Parse("2006-01-02", createdAfter); err == nil {
			filter.CreatedAfter = &t
		}
	}

	if createdBefore := c.Query("created_before"); createdBefore != "" {
		if t, err := time.Parse("2006-01-02", createdBefore); err == nil {
			filter.CreatedBefore = &t
		}
	}

	// Get transactions
	transactions, total, err := h.adminService.GetTransactions(filter)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(transactions, total, params)
	utils.PaginatedResponse(c, result)
}

// POST /admin/transactions/:id/refund
func (h *AdminHandler) ProcessRefund(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	transactionID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid transaction ID", nil)
		return
	}

	adminIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid admin ID", nil)
		return
	}

	var req struct {
		Reason string `json:"reason" validate:"required"`
	}

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
	if err := h.adminService.ProcessRefund(transactionID, adminID, req.Reason); err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, "Transaction not found")
			return
		}
		if strings.Contains(err.Error(), "can only refund") {
			utils.BadRequestResponse(c, err.Error(), nil)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyPaymentRefunded),
	})
}

// GET /admin/analytics
func (h *AdminHandler) GetAnalytics(c *gin.Context) {
	// Parse date range
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	metricsStr := c.Query("metrics")

	if startDateStr == "" || endDateStr == "" {
		utils.BadRequestResponse(c, "start_date and end_date are required", nil)
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid start_date format (YYYY-MM-DD)", nil)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid end_date format (YYYY-MM-DD)", nil)
		return
	}

	// Parse metrics
	var metrics []string
	if metricsStr != "" {
		metrics = strings.Split(metricsStr, ",")
	} else {
		metrics = []string{"user_registrations", "ip_creations", "license_applications", "product_sales", "revenue"}
	}

	// Get analytics
	analytics, err := h.adminService.GetAnalytics(startDate, endDate, metrics)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"analytics":  analytics,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
		"metrics":    metrics,
	})
}

// GET /admin/settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings, err := h.adminService.GetSettings()
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"settings": settings,
	})
}

// PUT /admin/settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	adminIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid admin ID", nil)
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Update each setting
	for key, value := range req {
		parts := strings.Split(key, ".")
		if len(parts) != 2 {
			continue // Skip invalid keys
		}

		category := parts[0]
		settingKey := parts[1]

		// Determine data type
		var dataType string
		switch value.(type) {
		case bool:
			dataType = "boolean"
		case float64:
			dataType = "float"
		case string:
			dataType = "string"
		default:
			dataType = "json"
		}

		if err := h.adminService.UpdateSetting(category, settingKey, value, dataType, adminID); err != nil {
			utils.BadRequestResponse(c, err.Error(), nil)
			return
		}
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyAdminSettingsUpdated),
	})
}

// GET /admin/reports/content
func (h *AdminHandler) GetContentReports(c *gin.Context) {
	params := utils.GetPaginationParams(c)

	// Get content reports
	reports, total, err := h.adminService.GetContentReports(params)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(reports, total, params)
	utils.PaginatedResponse(c, result)
}

// PUT /admin/reports/:id/resolve
func (h *AdminHandler) ResolveContentReport(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	reportID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid report ID", nil)
		return
	}

	adminIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid admin ID", nil)
		return
	}

	var req struct {
		Action string `json:"action" validate:"required"`
		Notes  string `json:"notes,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Resolve content report
	if err := h.adminService.ResolveContentReport(reportID, adminID, req.Action, req.Notes); err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, "Content report not found")
			return
		}
		if strings.Contains(err.Error(), "already resolved") {
			utils.BadRequestResponse(c, err.Error(), nil)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyAdminActionSuccess),
	})
}
