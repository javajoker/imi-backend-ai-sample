// internal/handlers/ip_asset.go
package handlers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javajoker/imi-backend/internal/i18n"
	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/services"
	"github.com/javajoker/imi-backend/internal/utils"
)

type IPAssetHandler struct {
	ipService      *services.IPService
	storageService *services.StorageService
}

func NewIPAssetHandler(ipService *services.IPService, storageService *services.StorageService) *IPAssetHandler {
	return &IPAssetHandler{
		ipService:      ipService,
		storageService: storageService,
	}
}

// GET /ip-assets
func (h *IPAssetHandler) GetIPAssets(c *gin.Context) {
	params := utils.GetPaginationParams(c)

	// Build search parameters
	searchParams := services.IPSearchParams{
		PaginationParams: params,
	}

	// Parse additional filters
	if category := c.Query("category"); category != "" {
		searchParams.Category = category
	}

	if verificationStatus := c.Query("verification_status"); verificationStatus != "" {
		status := models.VerificationStatus(verificationStatus)
		searchParams.VerificationStatus = &status
	}

	if creatorIDStr := c.Query("creator_id"); creatorIDStr != "" {
		if creatorID, err := uuid.Parse(creatorIDStr); err == nil {
			searchParams.CreatorID = &creatorID
		}
	}

	if tags := c.Query("tags"); tags != "" {
		searchParams.Tags = strings.Split(tags, ",")
	}

	// Search IP assets
	ipAssets, total, err := h.ipService.SearchIPAssets(searchParams)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(ipAssets, total, params)
	utils.PaginatedResponse(c, result)
}

// POST /ip-assets
func (h *IPAssetHandler) CreateIPAsset(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	creatorID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.CreateIPAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Create IP asset
	ipAsset, err := h.ipService.CreateIPAsset(creatorID, &req, nil) // File URLs would come from separate upload endpoint
	if err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"message":  i18n.T(lang, i18n.KeyIPAssetCreated),
		"ip_asset": ipAsset,
	})
}

// GET /ip-assets/:id
func (h *IPAssetHandler) GetIPAsset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid IP asset ID", nil)
		return
	}

	// Get current user ID if authenticated
	var userID *uuid.UUID
	if userIDStr, exists := utils.GetUserIDFromContext(c); exists {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			userID = &uid
		}
	}

	// Get IP asset
	ipAsset, err := h.ipService.GetIPAsset(id, userID)
	if err != nil {
		utils.NotFoundResponse(c, i18n.KeyIPAssetNotFound)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"ip_asset": ipAsset,
	})
}

// PUT /ip-assets/:id
func (h *IPAssetHandler) UpdateIPAsset(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid IP asset ID", nil)
		return
	}

	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	creatorID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.UpdateIPAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Update IP asset
	ipAsset, err := h.ipService.UpdateIPAsset(id, creatorID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyIPAssetNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message":  i18n.T(lang, i18n.KeyIPAssetUpdated),
		"ip_asset": ipAsset,
	})
}

// DELETE /ip-assets/:id
func (h *IPAssetHandler) DeleteIPAsset(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid IP asset ID", nil)
		return
	}

	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	creatorID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	// Delete IP asset
	if err := h.ipService.DeleteIPAsset(id, creatorID); err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyIPAssetNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyIPAssetDeleted),
	})
}

// GET /ip-assets/:id/licenses
func (h *IPAssetHandler) GetIPAssetLicenses(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid IP asset ID", nil)
		return
	}

	// Get license terms
	licenseTerms, err := h.ipService.GetLicenseTerms(id)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"license_terms": licenseTerms,
	})
}

// POST /ip-assets/:id/licenses
func (h *IPAssetHandler) CreateLicenseTerms(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	ipAssetID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid IP asset ID", nil)
		return
	}

	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	creatorID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.CreateLicenseTermsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Create license terms
	licenseTerms, err := h.ipService.CreateLicenseTerms(ipAssetID, creatorID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"message":       "License terms created successfully",
		"license_terms": licenseTerms,
	})
}

// POST /ip-assets/upload
func (h *IPAssetHandler) UploadFiles(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	_, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyFileUploadFailed), err.Error())
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		utils.BadRequestResponse(c, "No files uploaded", nil)
		return
	}

	var uploadedFiles []map[string]interface{}
	options := h.storageService.GetDefaultUploadOptions("ip_assets")

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}

		result, err := h.storageService.UploadFile(file, fileHeader, options)
		file.Close()

		if err != nil {
			continue
		}

		uploadedFiles = append(uploadedFiles, map[string]interface{}{
			"url":       result.URL,
			"key":       result.Key,
			"size":      result.Size,
			"mime_type": result.MimeType,
			"filename":  fileHeader.Filename,
		})
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyFileUploadSuccess),
		"files":   uploadedFiles,
	})
}

// GET /ip-assets/popular
func (h *IPAssetHandler) GetPopularIPAssets(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 50 {
		limit = 10
	}

	ipAssets, err := h.ipService.GetPopularIPAssets(limit)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"ip_assets": ipAssets,
	})
}

// GET /ip-assets/featured
func (h *IPAssetHandler) GetFeaturedIPAssets(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 50 {
		limit = 10
	}

	ipAssets, err := h.ipService.GetFeaturedIPAssets(limit)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"ip_assets": ipAssets,
	})
}

// GET /ip-assets/:id/statistics
func (h *IPAssetHandler) GetIPAssetStatistics(c *gin.Context) {
	idStr := c.Param("id")
	ipAssetID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid IP asset ID", nil)
		return
	}

	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	creatorID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	// Get statistics
	stats, err := h.ipService.GetIPAssetStatistics(ipAssetID, creatorID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"statistics": stats,
	})
}
