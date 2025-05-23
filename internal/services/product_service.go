// internal/services/product_service.go
package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/utils"
)

type ProductService struct {
	db                   *gorm.DB
	authorizationService *AuthorizationService
	notificationService  *NotificationService
}

type CreateProductRequest struct {
	LicenseID      uuid.UUID              `json:"license_id" validate:"required"`
	Title          string                 `json:"title" validate:"required,min=3,max=255"`
	Description    string                 `json:"description" validate:"required,min=10"`
	Category       string                 `json:"category" validate:"required"`
	Price          float64                `json:"price" validate:"required,min=0.01"`
	InventoryCount int                    `json:"inventory_count" validate:"min=0"`
	Images         []string               `json:"images,omitempty"`
	Specifications map[string]interface{} `json:"specifications,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
}

type UpdateProductRequest struct {
	Title          string                 `json:"title,omitempty" validate:"omitempty,min=3,max=255"`
	Description    string                 `json:"description,omitempty" validate:"omitempty,min=10"`
	Category       string                 `json:"category,omitempty"`
	Price          float64                `json:"price,omitempty" validate:"omitempty,min=0.01"`
	InventoryCount int                    `json:"inventory_count,omitempty" validate:"omitempty,min=0"`
	Images         []string               `json:"images,omitempty"`
	Specifications map[string]interface{} `json:"specifications,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Status         models.ProductStatus   `json:"status,omitempty"`
}

type ProductSearchParams struct {
	utils.PaginationParams
	CreatorID *uuid.UUID            `json:"creator_id,omitempty"`
	LicenseID *uuid.UUID            `json:"license_id,omitempty"`
	Status    *models.ProductStatus `json:"status,omitempty"`
	PriceMin  *float64              `json:"price_min,omitempty"`
	PriceMax  *float64              `json:"price_max,omitempty"`
	Tags      []string              `json:"tags,omitempty"`
	InStock   *bool                 `json:"in_stock,omitempty"`
}

type PurchaseProductRequest struct {
	Quantity      int                    `json:"quantity" validate:"required,min=1"`
	PaymentMethod string                 `json:"payment_method" validate:"required"`
	ShippingInfo  map[string]interface{} `json:"shipping_info,omitempty"`
	Notes         string                 `json:"notes,omitempty"`
}

func NewProductService(db *gorm.DB, authorizationService *AuthorizationService, notificationService *NotificationService) *ProductService {
	return &ProductService{
		db:                   db,
		authorizationService: authorizationService,
		notificationService:  notificationService,
	}
}

func (s *ProductService) CreateProduct(creatorID uuid.UUID, req *CreateProductRequest) (*models.Product, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify creator exists and is active
	var creator models.User
	if err := s.db.First(&creator, creatorID).Error; err != nil {
		return nil, fmt.Errorf("creator not found: %w", err)
	}

	if creator.Status != models.UserStatusActive {
		return nil, errors.New("creator account is not active")
	}

	if creator.UserType != models.UserTypeSecondaryCreator && creator.UserType != models.UserTypeCreator {
		return nil, errors.New("only secondary creators and creators can create products")
	}

	// Verify license ownership and validity
	var license models.LicenseApplication
	if err := s.db.Preload("IPAsset").Preload("LicenseTerms").
		First(&license, req.LicenseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("license not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify license belongs to the creator
	if license.ApplicantID != creatorID {
		return nil, errors.New("unauthorized to use this license")
	}

	// Verify license is approved and active
	if license.Status != models.ApplicationStatusApproved || !license.IsActive {
		return nil, errors.New("license is not active or approved")
	}

	// Check license expiration
	if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("license has expired")
	}

	// Create product
	product := &models.Product{
		CreatorID:            creatorID,
		LicenseID:            req.LicenseID,
		Title:                req.Title,
		Description:          req.Description,
		Category:             req.Category,
		Price:                req.Price,
		InventoryCount:       req.InventoryCount,
		Images:               req.Images,
		Specifications:       models.JSONB(req.Specifications),
		Tags:                 req.Tags,
		Status:               models.ProductStatusDraft,
		AuthenticityVerified: true, // Always true for licensed products
	}

	// Save product
	if err := s.db.Create(product).Error; err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Load relationships
	s.db.Preload("Creator").Preload("License").Preload("License.IPAsset").First(product, product.ID)

	// Create authorization chain record
	go s.createAuthorizationChain(product)

	return product, nil
}

func (s *ProductService) GetProduct(id uuid.UUID, userID *uuid.UUID) (*models.Product, error) {
	var product models.Product
	query := s.db.Preload("Creator").Preload("License").Preload("License.IPAsset").Preload("AuthChain")

	if err := query.First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check visibility permissions
	if product.Status != models.ProductStatusActive {
		// Only creator and admins can see non-active products
		if userID == nil || *userID != product.CreatorID {
			var user models.User
			if userID != nil {
				if err := s.db.First(&user, *userID).Error; err != nil || user.UserType != models.UserTypeAdmin {
					return nil, errors.New("product not found")
				}
			} else {
				return nil, errors.New("product not found")
			}
		}
	}

	// Increment view count if not the creator viewing
	if userID == nil || *userID != product.CreatorID {
		go s.incrementViewCount(id)
	}

	return &product, nil
}

func (s *ProductService) UpdateProduct(id uuid.UUID, creatorID uuid.UUID, req *UpdateProductRequest) (*models.Product, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Find and verify ownership
	var product models.Product
	if err := s.db.First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if product.CreatorID != creatorID {
		return nil, errors.New("unauthorized to update this product")
	}

	// Prepare updates
	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Price > 0 {
		updates["price"] = req.Price
	}
	if req.InventoryCount >= 0 {
		updates["inventory_count"] = req.InventoryCount
	}
	if req.Images != nil {
		updates["images"] = req.Images
	}
	if req.Specifications != nil {
		updates["specifications"] = models.JSONB(req.Specifications)
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	// Apply updates
	if err := s.db.Model(&product).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Reload with relationships
	s.db.Preload("Creator").Preload("License").Preload("License.IPAsset").First(&product, id)

	return &product, nil
}

func (s *ProductService) DeleteProduct(id uuid.UUID, creatorID uuid.UUID) error {
	// Find and verify ownership
	var product models.Product
	if err := s.db.First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("product not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if product.CreatorID != creatorID {
		return errors.New("unauthorized to delete this product")
	}

	// Check if product has been sold
	var salesCount int64
	if err := s.db.Model(&models.Transaction{}).
		Where("product_id = ? AND status = ?", id, models.TransactionStatusCompleted).
		Count(&salesCount).Error; err != nil {
		return fmt.Errorf("failed to check sales: %w", err)
	}

	if salesCount > 0 {
		return errors.New("cannot delete product with completed sales")
	}

	// Soft delete
	if err := s.db.Delete(&product).Error; err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	return nil
}

func (s *ProductService) SearchProducts(params ProductSearchParams) ([]models.Product, int64, error) {
	query := s.db.Model(&models.Product{}).
		Preload("Creator").Preload("License").Preload("License.IPAsset")

	// Apply filters
	if params.CreatorID != nil {
		query = query.Where("creator_id = ?", *params.CreatorID)
	}

	if params.LicenseID != nil {
		query = query.Where("license_id = ?", *params.LicenseID)
	}

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	} else {
		// Default to active products only
		query = query.Where("status = ?", models.ProductStatusActive)
	}

	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}

	if params.Search != "" {
		searchTerm := "%" + strings.ToLower(params.Search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	if params.PriceMin != nil {
		query = query.Where("price >= ?", *params.PriceMin)
	}

	if params.PriceMax != nil {
		query = query.Where("price <= ?", *params.PriceMax)
	}

	if len(params.Tags) > 0 {
		query = query.Where("tags && ?", params.Tags)
	}

	if params.InStock != nil && *params.InStock {
		query = query.Where("inventory_count > 0")
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	// Apply sorting
	allowedSortFields := []string{"created_at", "updated_at", "title", "price", "sales_count", "rating"}
	query = utils.ApplySort(query, params.PaginationParams, allowedSortFields)

	// Apply pagination
	query = utils.ApplyPagination(query, params.PaginationParams)

	// Execute query
	var products []models.Product
	if err := query.Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch products: %w", err)
	}

	return products, total, nil
}

func (s *ProductService) PurchaseProduct(productID uuid.UUID, buyerID uuid.UUID, req *PurchaseProductRequest) (*models.Transaction, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Start transaction
	return s.purchaseProductTransaction(productID, buyerID, req)
}

func (s *ProductService) purchaseProductTransaction(productID uuid.UUID, buyerID uuid.UUID, req *PurchaseProductRequest) (*models.Transaction, error) {
	var transaction *models.Transaction

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock and get product
		var product models.Product
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Preload("Creator").Preload("License").Preload("License.IPAsset").Preload("License.LicenseTerms").
			First(&product, productID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("product not found")
			}
			return fmt.Errorf("database error: %w", err)
		}

		// Verify product is active and available
		if product.Status != models.ProductStatusActive {
			return errors.New("product is not available for purchase")
		}

		if product.InventoryCount < req.Quantity {
			return errors.New("insufficient inventory")
		}

		// Verify buyer exists and is active
		var buyer models.User
		if err := tx.First(&buyer, buyerID).Error; err != nil {
			return fmt.Errorf("buyer not found: %w", err)
		}

		if buyer.Status != models.UserStatusActive {
			return errors.New("buyer account is not active")
		}

		// Calculate amounts
		totalAmount := product.Price * float64(req.Quantity)
		platformFeePercent := 5.0 // This should come from settings
		platformFee := totalAmount * (platformFeePercent / 100)

		// Calculate revenue shares
		revenueShares := s.calculateRevenueShares(totalAmount, platformFee, &product)

		// Create transaction
		transaction = &models.Transaction{
			TransactionType: models.TransactionTypeProductSale,
			BuyerID:         buyerID,
			SellerID:        product.CreatorID,
			ProductID:       &productID,
			Amount:          totalAmount,
			PlatformFee:     platformFee,
			RevenueShares:   models.JSONB(revenueShares),
			PaymentMethod:   req.PaymentMethod,
			Status:          models.TransactionStatusPending,
		}

		// Add shipping info and notes to transaction metadata
		if req.ShippingInfo != nil || req.Notes != "" {
			metadata := make(map[string]interface{})
			if req.ShippingInfo != nil {
				metadata["shipping_info"] = req.ShippingInfo
			}
			if req.Notes != "" {
				metadata["notes"] = req.Notes
			}
			metadata["quantity"] = req.Quantity
			transaction.RevenueShares["metadata"] = metadata
		}

		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Update product inventory
		if err := tx.Model(&product).UpdateColumn("inventory_count",
			gorm.Expr("inventory_count - ?", req.Quantity)).Error; err != nil {
			return fmt.Errorf("failed to update inventory: %w", err)
		}

		// Update sales count
		if err := tx.Model(&product).UpdateColumn("sales_count",
			gorm.Expr("sales_count + ?", req.Quantity)).Error; err != nil {
			return fmt.Errorf("failed to update sales count: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Process payment asynchronously
	go s.processPayment(transaction)

	// Load full transaction data
	s.db.Preload("Buyer").Preload("Seller").Preload("Product").First(transaction, transaction.ID)

	return transaction, nil
}

func (s *ProductService) calculateRevenueShares(totalAmount, platformFee float64, product *models.Product) map[string]interface{} {
	netAmount := totalAmount - platformFee

	// Get revenue share percentage from license terms
	revenueSharePercent := product.License.LicenseTerms.RevenueSharePercentage

	// Calculate shares
	ipCreatorShare := netAmount * (revenueSharePercent / 100)
	secondaryCreatorShare := netAmount - ipCreatorShare

	return map[string]interface{}{
		"total_amount":            totalAmount,
		"platform_fee":            platformFee,
		"net_amount":              netAmount,
		"ip_creator_share":        ipCreatorShare,
		"secondary_creator_share": secondaryCreatorShare,
		"ip_creator_id":           product.License.IPAsset.CreatorID,
		"secondary_creator_id":    product.CreatorID,
		"revenue_share_percent":   revenueSharePercent,
	}
}

func (s *ProductService) processPayment(transaction *models.Transaction) {
	// TODO: This would integrate with actual payment processors
	// For now, we'll simulate successful payment
	time.Sleep(2 * time.Second)

	now := time.Now()
	transaction.Status = models.TransactionStatusCompleted
	transaction.ProcessedAt = &now

	s.db.Save(transaction)

	// Send notifications
	if s.notificationService != nil {
		s.notificationService.SendPurchaseConfirmationNotification(transaction)
		s.notificationService.SendSaleNotification(transaction)
	}
}

func (s *ProductService) GetCreatorProducts(creatorID uuid.UUID, params utils.PaginationParams) ([]models.Product, int64, error) {
	query := s.db.Model(&models.Product{}).Where("creator_id = ?", creatorID).
		Preload("License").Preload("License.IPAsset")

	// Apply search if provided
	if params.Search != "" {
		searchTerm := "%" + strings.ToLower(params.Search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count creator products: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "updated_at", "title", "status", "sales_count"}
	query = utils.ApplySort(query, params, allowedSortFields)
	query = utils.ApplyPagination(query, params)

	// Execute query
	var products []models.Product
	if err := query.Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch creator products: %w", err)
	}

	return products, total, nil
}

func (s *ProductService) GetPopularProducts(limit int) ([]models.Product, error) {
	var products []models.Product
	if err := s.db.Where("status = ?", models.ProductStatusActive).
		Order("sales_count DESC, rating DESC, view_count DESC").
		Limit(limit).
		Preload("Creator").Preload("License").Preload("License.IPAsset").
		Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch popular products: %w", err)
	}

	return products, nil
}

func (s *ProductService) GetFeaturedProducts(limit int) ([]models.Product, error) {
	var products []models.Product
	if err := s.db.Where("status = ?", models.ProductStatusActive).
		Where("specifications->>'featured' = 'true'").
		Order("created_at DESC").
		Limit(limit).
		Preload("Creator").Preload("License").Preload("License.IPAsset").
		Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch featured products: %w", err)
	}

	return products, nil
}

func (s *ProductService) VerifyProductAuthenticity(productID uuid.UUID) (*models.AuthorizationChain, error) {
	var authChain models.AuthorizationChain
	if err := s.db.Where("product_id = ? AND is_active = ?", productID, true).
		Preload("Product").Preload("IPAsset").Preload("License").
		First(&authChain).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("authorization chain not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify the chain is valid
	if s.authorizationService != nil {
		if valid, err := s.authorizationService.VerifyChain(authChain.ID); err != nil || !valid {
			return nil, errors.New("authorization chain verification failed")
		}
	}

	return &authChain, nil
}

func (s *ProductService) GetProductStatistics(productID uuid.UUID, creatorID uuid.UUID) (map[string]interface{}, error) {
	// Verify ownership
	var product models.Product
	if err := s.db.First(&product, productID).Error; err != nil {
		return nil, errors.New("product not found")
	}

	if product.CreatorID != creatorID {
		return nil, errors.New("unauthorized to view statistics")
	}

	// Get sales statistics
	var salesStats struct {
		TotalSales    int64   `json:"total_sales"`
		TotalRevenue  float64 `json:"total_revenue"`
		AvgOrderValue float64 `json:"avg_order_value"`
	}

	s.db.Model(&models.Transaction{}).
		Where("product_id = ? AND status = ?", productID, models.TransactionStatusCompleted).
		Count(&salesStats.TotalSales)

	s.db.Model(&models.Transaction{}).
		Where("product_id = ? AND status = ?", productID, models.TransactionStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").Scan(&salesStats.TotalRevenue)

	if salesStats.TotalSales > 0 {
		salesStats.AvgOrderValue = salesStats.TotalRevenue / float64(salesStats.TotalSales)
	}

	return map[string]interface{}{
		"view_count":      product.ViewCount,
		"sales_count":     product.SalesCount,
		"rating":          product.Rating,
		"review_count":    product.ReviewCount,
		"inventory_count": product.InventoryCount,
		"sales_stats":     salesStats,
		"created_at":      product.CreatedAt,
		"updated_at":      product.UpdatedAt,
	}, nil
}

// Helper methods

func (s *ProductService) incrementViewCount(productID uuid.UUID) {
	s.db.Model(&models.Product{}).Where("id = ?", productID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1"))
}

func (s *ProductService) createAuthorizationChain(product *models.Product) {
	if s.authorizationService != nil {
		s.authorizationService.CreateProductAuthChain(product)
	}
}
