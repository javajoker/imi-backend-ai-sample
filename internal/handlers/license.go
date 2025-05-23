// internal/handlers/license.go
package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javajoker/imi-backend/internal/i18n"
	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/services"
	"github.com/javajoker/imi-backend/internal/utils"
)

type LicenseHandler struct {
	licenseService *services.LicenseService
}

func NewLicenseHandler(licenseService *services.LicenseService) *LicenseHandler {
	return &LicenseHandler{
		licenseService: licenseService,
	}
}

// POST /licenses/apply
func (h *LicenseHandler) ApplyForLicense(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	applicantID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.ApplyLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Apply for license
	application, err := h.licenseService.ApplyForLicense(applicantID, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"message":     i18n.T(lang, i18n.KeyLicenseApplied),
		"application": application,
	})
}

// GET /licenses/applications
func (h *LicenseHandler) GetLicenseApplications(c *gin.Context) {
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

	// Build search parameters
	searchParams := services.LicenseSearchParams{
		PaginationParams: params,
	}

	// Parse filters
	if ipAssetIDStr := c.Query("ip_asset_id"); ipAssetIDStr != "" {
		if ipAssetID, err := uuid.Parse(ipAssetIDStr); err == nil {
			searchParams.IPAssetID = &ipAssetID
		}
	}

	if status := c.Query("status"); status != "" {
		appStatus := models.ApplicationStatus(status)
		searchParams.Status = &appStatus
	}

	if licenseType := c.Query("license_type"); licenseType != "" {
		lType := models.LicenseType(licenseType)
		searchParams.LicenseType = &lType
	}

	// Search applications
	applications, total, err := h.licenseService.SearchLicenseApplications(searchParams, userID)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(applications, total, params)
	utils.PaginatedResponse(c, result)
}

// GET /licenses/:id
func (h *LicenseHandler) GetLicenseApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid license application ID", nil)
		return
	}

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

	// Get license application
	application, err := h.licenseService.GetLicenseApplication(id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyLicenseNotFound)
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"application": application,
	})
}

// PUT /licenses/:id/approve
func (h *LicenseHandler) ApproveLicense(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	applicationID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid license application ID", nil)
		return
	}

	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	approverID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.ApproveLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Approve license
	application, err := h.licenseService.ApproveLicense(applicationID, approverID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyLicenseNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message":     i18n.T(lang, i18n.KeyLicenseApproved),
		"application": application,
	})
}

// PUT /licenses/:id/reject
func (h *LicenseHandler) RejectLicense(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	applicationID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid license application ID", nil)
		return
	}

	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	rejecterID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.RejectLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Reject license
	application, err := h.licenseService.RejectLicense(applicationID, rejecterID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyLicenseNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message":     i18n.T(lang, i18n.KeyLicenseRejected),
		"application": application,
	})
}

// PUT /licenses/:id/revoke
func (h *LicenseHandler) RevokeLicense(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	applicationID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid license application ID", nil)
		return
	}

	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	revokerID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.RevokeLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Revoke license
	application, err := h.licenseService.RevokeLicense(applicationID, revokerID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyLicenseNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message":     i18n.T(lang, i18n.KeyLicenseRevoked),
		"application": application,
	})
}

// GET /licenses/:id/verify
func (h *LicenseHandler) VerifyLicense(c *gin.Context) {
	idStr := c.Param("id")
	licenseID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid license ID", nil)
		return
	}

	// Verify license
	license, err := h.licenseService.VerifyLicense(licenseID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyLicenseNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"valid":   true,
		"license": license,
	})
}

// GET /licenses/my-licenses
func (h *LicenseHandler) GetMyLicenses(c *gin.Context) {
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

	// Get user licenses
	licenses, total, err := h.licenseService.GetUserLicenses(userID, params)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(licenses, total, params)
	utils.PaginatedResponse(c, result)
}

// GET /licenses/statistics
func (h *LicenseHandler) GetLicenseStatistics(c *gin.Context) {
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

	// Get license statistics
	stats, err := h.licenseService.GetLicenseStatistics(userID)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"statistics": stats,
	})
}
