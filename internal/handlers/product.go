// internal/handlers/product.go
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

type ProductHandler struct {
	productService *services.ProductService
	storageService *services.StorageService
}

func NewProductHandler(productService *services.ProductService, storageService *services.StorageService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		storageService: storageService,
	}
}

// GET /products
func (h *ProductHandler) GetProducts(c *gin.Context) {
	params := utils.GetPaginationParams(c)

	// Build search parameters
	searchParams := services.ProductSearchParams{
		PaginationParams: params,
	}

	// Parse additional filters
	if category := c.Query("category"); category != "" {
		searchParams.Category = category
	}

	if status := c.Query("status"); status != "" {
		productStatus := models.ProductStatus(status)
		searchParams.Status = &productStatus
	}

	if creatorIDStr := c.Query("creator_id"); creatorIDStr != "" {
		if creatorID, err := uuid.Parse(creatorIDStr); err == nil {
			searchParams.CreatorID = &creatorID
		}
	}

	if licenseIDStr := c.Query("license_id"); licenseIDStr != "" {
		if licenseID, err := uuid.Parse(licenseIDStr); err == nil {
			searchParams.LicenseID = &licenseID
		}
	}

	if priceMinStr := c.Query("price_min"); priceMinStr != "" {
		if priceMin, err := strconv.ParseFloat(priceMinStr, 64); err == nil {
			searchParams.PriceMin = &priceMin
		}
	}

	if priceMaxStr := c.Query("price_max"); priceMaxStr != "" {
		if priceMax, err := strconv.ParseFloat(priceMaxStr, 64); err == nil {
			searchParams.PriceMax = &priceMax
		}
	}

	if tags := c.Query("tags"); tags != "" {
		searchParams.Tags = strings.Split(tags, ",")
	}

	if inStockStr := c.Query("in_stock"); inStockStr != "" {
		if inStock, err := strconv.ParseBool(inStockStr); err == nil {
			searchParams.InStock = &inStock
		}
	}

	// Search products
	products, total, err := h.productService.SearchProducts(searchParams)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	result := utils.CreatePaginationResult(products, total, params)
	utils.PaginatedResponse(c, result)
}

// POST /products
func (h *ProductHandler) CreateProduct(c *gin.Context) {
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

	var req services.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Create product
	product, err := h.productService.CreateProduct(creatorID, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyProductCreated),
		"product": product,
	})
}

// GET /products/:id
func (h *ProductHandler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid product ID", nil)
		return
	}

	// Get current user ID if authenticated
	var userID *uuid.UUID
	if userIDStr, exists := utils.GetUserIDFromContext(c); exists {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			userID = &uid
		}
	}

	// Get product
	product, err := h.productService.GetProduct(id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyProductNotFound)
			return
		}
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"product": product,
	})
}

// PUT /products/:id
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid product ID", nil)
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

	var req services.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Update product
	product, err := h.productService.UpdateProduct(id, creatorID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyProductNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyProductUpdated),
		"product": product,
	})
}

// DELETE /products/:id
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid product ID", nil)
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

	// Delete product
	if err := h.productService.DeleteProduct(id, creatorID); err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyProductNotFound)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyProductDeleted),
	})
}

// POST /products/:id/purchase
func (h *ProductHandler) PurchaseProduct(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	idStr := c.Param("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid product ID", nil)
		return
	}

	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	buyerID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req services.PurchaseProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Purchase product
	transaction, err := h.productService.PurchaseProduct(productID, buyerID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, i18n.KeyProductNotFound)
			return
		}
		if strings.Contains(err.Error(), "insufficient") {
			utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyProductOutOfStock), nil)
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"message":     i18n.T(lang, i18n.KeyProductPurchased),
		"transaction": transaction,
	})
}

// GET /products/:id/verify
func (h *ProductHandler) VerifyProduct(c *gin.Context) {
	idStr := c.Param("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid product ID", nil)
		return
	}

	// Verify product authenticity
	authChain, err := h.productService.VerifyProductAuthenticity(productID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFoundResponse(c, "Authorization chain not found")
			return
		}
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"verified":            true,
		"authorization_chain": authChain,
	})
}

// GET /products/popular
func (h *ProductHandler) GetPopularProducts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 50 {
		limit = 10
	}

	products, err := h.productService.GetPopularProducts(limit)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"products": products,
	})
}

// GET /products/featured
func (h *ProductHandler) GetFeaturedProducts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 50 {
		limit = 10
	}

	products, err := h.productService.GetFeaturedProducts(limit)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"products": products,
	})
}

// POST /products/upload-images
func (h *ProductHandler) UploadProductImages(c *gin.Context) {
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

	files := form.File["images"]
	if len(files) == 0 {
		utils.BadRequestResponse(c, "No images uploaded", nil)
		return
	}

	var uploadedImages []map[string]interface{}
	options := h.storageService.GetDefaultUploadOptions("products")

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}

		// Validate image
		if err := h.storageService.ValidateImage(file); err != nil {
			file.Close()
			continue
		}

		result, err := h.storageService.UploadFile(file, fileHeader, options)
		file.Close()

		if err != nil {
			continue
		}

		uploadedImages = append(uploadedImages, map[string]interface{}{
			"url":       result.URL,
			"key":       result.Key,
			"size":      result.Size,
			"mime_type": result.MimeType,
		})
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyFileUploadSuccess),
		"images":  uploadedImages,
	})
}

// GET /products/:id/statistics
func (h *ProductHandler) GetProductStatistics(c *gin.Context) {
	idStr := c.Param("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid product ID", nil)
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
	stats, err := h.productService.GetProductStatistics(productID, creatorID)
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
