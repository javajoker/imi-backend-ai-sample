// internal/services/ip_service.go
package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/utils"
)

type IPService struct {
	db                *gorm.DB
	blockchainService *BlockchainService
	storageService    *StorageService
}

type CreateIPAssetRequest struct {
	Title       string                 `json:"title" validate:"required,min=3,max=255"`
	Description string                 `json:"description" validate:"required,min=10"`
	Category    string                 `json:"category" validate:"required"`
	ContentType string                 `json:"content_type" validate:"required"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateIPAssetRequest struct {
	Title       string                 `json:"title,omitempty" validate:"omitempty,min=3,max=255"`
	Description string                 `json:"description,omitempty" validate:"omitempty,min=10"`
	Category    string                 `json:"category,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CreateLicenseTermsRequest struct {
	LicenseType            models.LicenseType `json:"license_type" validate:"required"`
	RevenueSharePercentage float64            `json:"revenue_share_percentage" validate:"required,min=5,max=50"`
	BaseFee                float64            `json:"base_fee,omitempty" validate:"min=0"`
	Territory              string             `json:"territory,omitempty"`
	Duration               string             `json:"duration,omitempty"`
	Requirements           string             `json:"requirements,omitempty"`
	Restrictions           string             `json:"restrictions,omitempty"`
	AutoApprove            bool               `json:"auto_approve,omitempty"`
	MaxLicenses            int                `json:"max_licenses,omitempty" validate:"min=0"`
}

type IPSearchParams struct {
	utils.PaginationParams
	CreatorID          *uuid.UUID                 `json:"creator_id,omitempty"`
	VerificationStatus *models.VerificationStatus `json:"verification_status,omitempty"`
	Status             *models.ProductStatus      `json:"status,omitempty"`
	PriceMin           *float64                   `json:"price_min,omitempty"`
	PriceMax           *float64                   `json:"price_max,omitempty"`
	Tags               []string                   `json:"tags,omitempty"`
}

func NewIPService(db *gorm.DB, blockchainService *BlockchainService, storageService *StorageService) *IPService {
	return &IPService{
		db:                db,
		blockchainService: blockchainService,
		storageService:    storageService,
	}
}

func (s *IPService) CreateIPAsset(creatorID uuid.UUID, req *CreateIPAssetRequest, fileURLs []string) (*models.IPAsset, error) {
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

	if creator.UserType != models.UserTypeCreator && creator.UserType != models.UserTypeAdmin {
		return nil, errors.New("only creators can create IP assets")
	}

	// Create IP asset
	ipAsset := &models.IPAsset{
		CreatorID:          creatorID,
		Title:              req.Title,
		Description:        req.Description,
		Category:           req.Category,
		ContentType:        req.ContentType,
		FileURLs:           fileURLs,
		Tags:               req.Tags,
		Metadata:           models.JSONB(req.Metadata),
		VerificationStatus: models.VerificationStatusPending,
		Status:             models.ProductStatusActive,
	}

	// Save to database
	if err := s.db.Create(ipAsset).Error; err != nil {
		return nil, fmt.Errorf("failed to create IP asset: %w", err)
	}

	// Load creator relationship
	s.db.Preload("Creator").First(ipAsset, ipAsset.ID)

	// Create blockchain record asynchronously
	go s.createBlockchainRecord(ipAsset)

	return ipAsset, nil
}

func (s *IPService) GetIPAsset(id uuid.UUID, userID *uuid.UUID) (*models.IPAsset, error) {
	var ipAsset models.IPAsset
	query := s.db.Preload("Creator").Preload("LicenseTerms")

	if err := query.First(&ipAsset, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("IP asset not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check visibility permissions
	if ipAsset.Status != models.ProductStatusActive &&
		(userID == nil || *userID != ipAsset.CreatorID) {
		return nil, errors.New("IP asset not found")
	}

	// Increment view count if not the creator viewing
	if userID == nil || *userID != ipAsset.CreatorID {
		go s.incrementViewCount(id)
	}

	return &ipAsset, nil
}

func (s *IPService) UpdateIPAsset(id uuid.UUID, creatorID uuid.UUID, req *UpdateIPAssetRequest) (*models.IPAsset, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Find and verify ownership
	var ipAsset models.IPAsset
	if err := s.db.First(&ipAsset, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("IP asset not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if ipAsset.CreatorID != creatorID {
		return nil, errors.New("unauthorized to update this IP asset")
	}

	// Update fields
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
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.Metadata != nil {
		updates["metadata"] = models.JSONB(req.Metadata)
	}

	// Reset verification status if content changed
	if req.Title != "" || req.Description != "" {
		updates["verification_status"] = models.VerificationStatusPending
	}

	if err := s.db.Model(&ipAsset).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update IP asset: %w", err)
	}

	// Reload with relationships
	s.db.Preload("Creator").Preload("LicenseTerms").First(&ipAsset, id)

	return &ipAsset, nil
}

func (s *IPService) DeleteIPAsset(id uuid.UUID, creatorID uuid.UUID) error {
	// Find and verify ownership
	var ipAsset models.IPAsset
	if err := s.db.First(&ipAsset, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("IP asset not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if ipAsset.CreatorID != creatorID {
		return errors.New("unauthorized to delete this IP asset")
	}

	// Check if there are active licenses
	var licenseCount int64
	if err := s.db.Model(&models.LicenseApplication{}).
		Where("ip_asset_id = ? AND status = ?", id, models.ApplicationStatusApproved).
		Count(&licenseCount).Error; err != nil {
		return fmt.Errorf("failed to check licenses: %w", err)
	}

	if licenseCount > 0 {
		return errors.New("cannot delete IP asset with active licenses")
	}

	// Soft delete
	if err := s.db.Delete(&ipAsset).Error; err != nil {
		return fmt.Errorf("failed to delete IP asset: %w", err)
	}

	return nil
}

func (s *IPService) SearchIPAssets(params IPSearchParams) ([]models.IPAsset, int64, error) {
	query := s.db.Model(&models.IPAsset{}).Preload("Creator")

	// Apply filters
	if params.CreatorID != nil {
		query = query.Where("creator_id = ?", *params.CreatorID)
	}

	if params.VerificationStatus != nil {
		query = query.Where("verification_status = ?", *params.VerificationStatus)
	}

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	} else {
		// Default to active assets only
		query = query.Where("status = ?", models.ProductStatusActive)
	}

	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}

	if params.Search != "" {
		searchTerm := "%" + strings.ToLower(params.Search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	if len(params.Tags) > 0 {
		query = query.Where("tags && ?", params.Tags)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count IP assets: %w", err)
	}

	// Apply sorting
	allowedSortFields := []string{"created_at", "updated_at", "title", "view_count", "like_count"}
	query = utils.ApplySort(query, params.PaginationParams, allowedSortFields)

	// Apply pagination
	query = utils.ApplyPagination(query, params.PaginationParams)

	// Execute query
	var ipAssets []models.IPAsset
	if err := query.Find(&ipAssets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch IP assets: %w", err)
	}

	return ipAssets, total, nil
}

func (s *IPService) CreateLicenseTerms(ipAssetID uuid.UUID, creatorID uuid.UUID, req *CreateLicenseTermsRequest) (*models.LicenseTerms, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify IP asset ownership
	var ipAsset models.IPAsset
	if err := s.db.First(&ipAsset, ipAssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("IP asset not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if ipAsset.CreatorID != creatorID {
		return nil, errors.New("unauthorized to create license terms for this IP asset")
	}

	if ipAsset.VerificationStatus != models.VerificationStatusApproved {
		return nil, errors.New("IP asset must be approved before creating license terms")
	}

	// Set defaults
	territory := req.Territory
	if territory == "" {
		territory = "global"
	}

	duration := req.Duration
	if duration == "" {
		duration = "perpetual"
	}

	// Create license terms
	licenseTerms := &models.LicenseTerms{
		IPAssetID:              ipAssetID,
		LicenseType:            req.LicenseType,
		RevenueSharePercentage: req.RevenueSharePercentage,
		BaseFee:                req.BaseFee,
		Territory:              territory,
		Duration:               duration,
		Requirements:           req.Requirements,
		Restrictions:           req.Restrictions,
		AutoApprove:            req.AutoApprove,
		MaxLicenses:            req.MaxLicenses,
		IsActive:               true,
	}

	if err := s.db.Create(licenseTerms).Error; err != nil {
		return nil, fmt.Errorf("failed to create license terms: %w", err)
	}

	// Load relationships
	s.db.Preload("IPAsset").First(licenseTerms, licenseTerms.ID)

	return licenseTerms, nil
}

func (s *IPService) GetLicenseTerms(ipAssetID uuid.UUID) ([]models.LicenseTerms, error) {
	var licenseTerms []models.LicenseTerms
	if err := s.db.Where("ip_asset_id = ? AND is_active = ?", ipAssetID, true).
		Find(&licenseTerms).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch license terms: %w", err)
	}

	return licenseTerms, nil
}

func (s *IPService) UpdateLicenseTerms(id uuid.UUID, creatorID uuid.UUID, req *CreateLicenseTermsRequest) (*models.LicenseTerms, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Find license terms and verify ownership
	var licenseTerms models.LicenseTerms
	if err := s.db.Preload("IPAsset").First(&licenseTerms, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("license terms not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if licenseTerms.IPAsset.CreatorID != creatorID {
		return nil, errors.New("unauthorized to update these license terms")
	}

	// Check if there are pending applications
	var pendingCount int64
	if err := s.db.Model(&models.LicenseApplication{}).
		Where("license_terms_id = ? AND status = ?", id, models.ApplicationStatusPending).
		Count(&pendingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check pending applications: %w", err)
	}

	if pendingCount > 0 {
		return nil, errors.New("cannot update license terms with pending applications")
	}

	// Update fields
	licenseTerms.LicenseType = req.LicenseType
	licenseTerms.RevenueSharePercentage = req.RevenueSharePercentage
	licenseTerms.BaseFee = req.BaseFee
	licenseTerms.Territory = req.Territory
	licenseTerms.Duration = req.Duration
	licenseTerms.Requirements = req.Requirements
	licenseTerms.Restrictions = req.Restrictions
	licenseTerms.AutoApprove = req.AutoApprove
	licenseTerms.MaxLicenses = req.MaxLicenses

	if err := s.db.Save(&licenseTerms).Error; err != nil {
		return nil, fmt.Errorf("failed to update license terms: %w", err)
	}

	return &licenseTerms, nil
}

func (s *IPService) GetCreatorIPAssets(creatorID uuid.UUID, params utils.PaginationParams) ([]models.IPAsset, int64, error) {
	query := s.db.Model(&models.IPAsset{}).Where("creator_id = ?", creatorID)

	// Apply search if provided
	if params.Search != "" {
		searchTerm := "%" + strings.ToLower(params.Search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count creator IP assets: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "updated_at", "title", "verification_status"}
	query = utils.ApplySort(query, params, allowedSortFields)
	query = utils.ApplyPagination(query, params)

	// Execute query
	var ipAssets []models.IPAsset
	if err := query.Preload("LicenseTerms").Find(&ipAssets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch creator IP assets: %w", err)
	}

	return ipAssets, total, nil
}

// Helper methods

func (s *IPService) incrementViewCount(ipAssetID uuid.UUID) {
	s.db.Model(&models.IPAsset{}).Where("id = ?", ipAssetID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1"))
}

func (s *IPService) createBlockchainRecord(ipAsset *models.IPAsset) {
	if s.blockchainService != nil {
		if hash, err := s.blockchainService.CreateIPRecord(ipAsset.ID, ipAsset.CreatorID); err == nil {
			s.db.Model(ipAsset).UpdateColumn("blockchain_hash", hash)
		}
	}
}

func (s *IPService) GetPopularIPAssets(limit int) ([]models.IPAsset, error) {
	var ipAssets []models.IPAsset
	if err := s.db.Where("status = ? AND verification_status = ?",
		models.ProductStatusActive, models.VerificationStatusApproved).
		Order("view_count DESC, like_count DESC").
		Limit(limit).
		Preload("Creator").
		Find(&ipAssets).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch popular IP assets: %w", err)
	}

	return ipAssets, nil
}

func (s *IPService) GetFeaturedIPAssets(limit int) ([]models.IPAsset, error) {
	var ipAssets []models.IPAsset
	if err := s.db.Where("status = ? AND verification_status = ?",
		models.ProductStatusActive, models.VerificationStatusApproved).
		Where("metadata->>'featured' = 'true'").
		Order("created_at DESC").
		Limit(limit).
		Preload("Creator").
		Find(&ipAssets).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch featured IP assets: %w", err)
	}

	return ipAssets, nil
}

func (s *IPService) GetIPAssetStatistics(ipAssetID uuid.UUID, creatorID uuid.UUID) (map[string]interface{}, error) {
	// Verify ownership
	var ipAsset models.IPAsset
	if err := s.db.First(&ipAsset, ipAssetID).Error; err != nil {
		return nil, errors.New("IP asset not found")
	}

	if ipAsset.CreatorID != creatorID {
		return nil, errors.New("unauthorized to view statistics")
	}

	// Get license statistics
	var licenseStats struct {
		TotalApplications int64 `json:"total_applications"`
		ApprovedLicenses  int64 `json:"approved_licenses"`
		PendingLicenses   int64 `json:"pending_licenses"`
		RejectedLicenses  int64 `json:"rejected_licenses"`
	}

	s.db.Model(&models.LicenseApplication{}).
		Where("ip_asset_id = ?", ipAssetID).
		Count(&licenseStats.TotalApplications)

	s.db.Model(&models.LicenseApplication{}).
		Where("ip_asset_id = ? AND status = ?", ipAssetID, models.ApplicationStatusApproved).
		Count(&licenseStats.ApprovedLicenses)

	s.db.Model(&models.LicenseApplication{}).
		Where("ip_asset_id = ? AND status = ?", ipAssetID, models.ApplicationStatusPending).
		Count(&licenseStats.PendingLicenses)

	s.db.Model(&models.LicenseApplication{}).
		Where("ip_asset_id = ? AND status = ?", ipAssetID, models.ApplicationStatusRejected).
		Count(&licenseStats.RejectedLicenses)

	// Get revenue statistics (placeholder - actual implementation would be more complex)
	var totalRevenue float64
	s.db.Model(&models.Transaction{}).
		Joins("JOIN products ON transactions.product_id = products.id").
		Joins("JOIN license_applications ON products.license_id = license_applications.id").
		Where("license_applications.ip_asset_id = ? AND transactions.status = ?",
			ipAssetID, models.TransactionStatusCompleted).
		Select("COALESCE(SUM(revenue_shares->>'ip_creator'), 0)").
		Scan(&totalRevenue)

	return map[string]interface{}{
		"view_count":    ipAsset.ViewCount,
		"like_count":    ipAsset.LikeCount,
		"license_stats": licenseStats,
		"total_revenue": totalRevenue,
		"created_at":    ipAsset.CreatedAt,
		"updated_at":    ipAsset.UpdatedAt,
	}, nil
}
