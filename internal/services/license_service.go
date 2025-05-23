// internal/services/license_service.go
package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/utils"
)

type LicenseService struct {
	db                  *gorm.DB
	notificationService *NotificationService
}

type ApplyLicenseRequest struct {
	IPAssetID       uuid.UUID              `json:"ip_asset_id" validate:"required"`
	LicenseTermsID  uuid.UUID              `json:"license_terms_id" validate:"required"`
	ApplicationData map[string]interface{} `json:"application_data,omitempty"`
	Message         string                 `json:"message,omitempty"`
}

type ApproveLicenseRequest struct {
	Message string `json:"message,omitempty"`
}

type RejectLicenseRequest struct {
	Reason  string `json:"reason" validate:"required"`
	Message string `json:"message,omitempty"`
}

type RevokeLicenseRequest struct {
	Reason  string `json:"reason" validate:"required"`
	Message string `json:"message,omitempty"`
}

type LicenseSearchParams struct {
	utils.PaginationParams
	ApplicantID *uuid.UUID                `json:"applicant_id,omitempty"`
	IPAssetID   *uuid.UUID                `json:"ip_asset_id,omitempty"`
	CreatorID   *uuid.UUID                `json:"creator_id,omitempty"`
	Status      *models.ApplicationStatus `json:"status,omitempty"`
	LicenseType *models.LicenseType       `json:"license_type,omitempty"`
}

func NewLicenseService(db *gorm.DB, notificationService *NotificationService) *LicenseService {
	return &LicenseService{
		db:                  db,
		notificationService: notificationService,
	}
}

func (s *LicenseService) ApplyForLicense(applicantID uuid.UUID, req *ApplyLicenseRequest) (*models.LicenseApplication, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify applicant exists and is eligible
	var applicant models.User
	if err := s.db.First(&applicant, applicantID).Error; err != nil {
		return nil, fmt.Errorf("applicant not found: %w", err)
	}

	if applicant.Status != models.UserStatusActive {
		return nil, errors.New("applicant account is not active")
	}

	if applicant.UserType != models.UserTypeSecondaryCreator && applicant.UserType != models.UserTypeCreator {
		return nil, errors.New("only secondary creators and creators can apply for licenses")
	}

	// Get IP asset and license terms
	var ipAsset models.IPAsset
	if err := s.db.Preload("Creator").First(&ipAsset, req.IPAssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("IP asset not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check if IP asset is approved
	if ipAsset.VerificationStatus != models.VerificationStatusApproved {
		return nil, errors.New("IP asset is not approved for licensing")
	}

	// Check if applicant is trying to license their own IP
	if ipAsset.CreatorID == applicantID {
		return nil, errors.New("cannot apply for license on your own IP asset")
	}

	// Get license terms
	var licenseTerms models.LicenseTerms
	if err := s.db.Where("id = ? AND ip_asset_id = ? AND is_active = ?",
		req.LicenseTermsID, req.IPAssetID, true).First(&licenseTerms).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("license terms not found or inactive")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check if applicant already has an active or pending application
	var existingApp models.LicenseApplication
	if err := s.db.Where("ip_asset_id = ? AND applicant_id = ? AND status IN (?, ?)",
		req.IPAssetID, applicantID, models.ApplicationStatusPending, models.ApplicationStatusApproved).
		First(&existingApp).Error; err == nil {
		if existingApp.Status == models.ApplicationStatusApproved {
			return nil, errors.New("you already have an approved license for this IP asset")
		}
		return nil, errors.New("you already have a pending application for this IP asset")
	}

	// Check license limit
	if licenseTerms.MaxLicenses > 0 {
		var approvedCount int64
		if err := s.db.Model(&models.LicenseApplication{}).
			Where("license_terms_id = ? AND status = ?", req.LicenseTermsID, models.ApplicationStatusApproved).
			Count(&approvedCount).Error; err != nil {
			return nil, fmt.Errorf("failed to check license count: %w", err)
		}

		if approvedCount >= int64(licenseTerms.MaxLicenses) {
			return nil, errors.New("license limit reached for this license terms")
		}
	}

	// Prepare application data
	applicationData := req.ApplicationData
	if applicationData == nil {
		applicationData = make(map[string]interface{})
	}
	if req.Message != "" {
		applicationData["message"] = req.Message
	}
	applicationData["applied_at"] = time.Now()

	// Create license application
	application := &models.LicenseApplication{
		IPAssetID:       req.IPAssetID,
		ApplicantID:     applicantID,
		LicenseTermsID:  req.LicenseTermsID,
		ApplicationData: models.JSONB(applicationData),
		Status:          models.ApplicationStatusPending,
		IsActive:        true,
	}

	// Auto-approve if enabled
	if licenseTerms.AutoApprove {
		application.Status = models.ApplicationStatusApproved
		now := time.Now()
		application.ApprovedAt = &now
		application.ApprovedBy = &ipAsset.CreatorID // Auto-approved by system
	}

	// Save application
	if err := s.db.Create(application).Error; err != nil {
		return nil, fmt.Errorf("failed to create license application: %w", err)
	}

	// Load relationships
	s.db.Preload("IPAsset").Preload("Applicant").Preload("LicenseTerms").First(application, application.ID)

	// Send notifications
	go s.sendApplicationNotifications(application, licenseTerms.AutoApprove)

	return application, nil
}

func (s *LicenseService) ApproveLicense(applicationID uuid.UUID, approverID uuid.UUID, req *ApproveLicenseRequest) (*models.LicenseApplication, error) {
	// Find application
	var application models.LicenseApplication
	if err := s.db.Preload("IPAsset").Preload("Applicant").Preload("LicenseTerms").
		First(&application, applicationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("license application not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify approver is the IP creator or admin
	if application.IPAsset.CreatorID != approverID {
		var approver models.User
		if err := s.db.First(&approver, approverID).Error; err != nil {
			return nil, errors.New("unauthorized to approve license")
		}
		if approver.UserType != models.UserTypeAdmin {
			return nil, errors.New("unauthorized to approve license")
		}
	}

	// Check if already processed
	if application.Status != models.ApplicationStatusPending {
		return nil, errors.New("license application already processed")
	}

	// Check license limit
	if application.LicenseTerms.MaxLicenses > 0 {
		var approvedCount int64
		if err := s.db.Model(&models.LicenseApplication{}).
			Where("license_terms_id = ? AND status = ? AND id != ?",
				application.LicenseTermsID, models.ApplicationStatusApproved, applicationID).
			Count(&approvedCount).Error; err != nil {
			return nil, fmt.Errorf("failed to check license count: %w", err)
		}

		if approvedCount >= int64(application.LicenseTerms.MaxLicenses) {
			return nil, errors.New("license limit reached")
		}
	}

	// Update application
	now := time.Now()
	application.Status = models.ApplicationStatusApproved
	application.ApprovedAt = &now
	application.ApprovedBy = &approverID

	// Update application data with approval message
	if req.Message != "" {
		if application.ApplicationData == nil {
			application.ApplicationData = make(models.JSONB)
		}
		application.ApplicationData["approval_message"] = req.Message
		application.ApplicationData["approved_at"] = now
	}

	if err := s.db.Save(&application).Error; err != nil {
		return nil, fmt.Errorf("failed to update license application: %w", err)
	}

	// Send notification to applicant
	go s.sendApprovalNotification(&application)

	return &application, nil
}

func (s *LicenseService) RejectLicense(applicationID uuid.UUID, rejecterID uuid.UUID, req *RejectLicenseRequest) (*models.LicenseApplication, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Find application
	var application models.LicenseApplication
	if err := s.db.Preload("IPAsset").Preload("Applicant").
		First(&application, applicationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("license application not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify rejecter is the IP creator or admin
	if application.IPAsset.CreatorID != rejecterID {
		var rejecter models.User
		if err := s.db.First(&rejecter, rejecterID).Error; err != nil {
			return nil, errors.New("unauthorized to reject license")
		}
		if rejecter.UserType != models.UserTypeAdmin {
			return nil, errors.New("unauthorized to reject license")
		}
	}

	// Check if already processed
	if application.Status != models.ApplicationStatusPending {
		return nil, errors.New("license application already processed")
	}

	// Update application
	application.Status = models.ApplicationStatusRejected
	application.RejectionReason = req.Reason

	// Update application data with rejection details
	if application.ApplicationData == nil {
		application.ApplicationData = make(models.JSONB)
	}
	application.ApplicationData["rejection_reason"] = req.Reason
	application.ApplicationData["rejection_message"] = req.Message
	application.ApplicationData["rejected_at"] = time.Now()
	application.ApplicationData["rejected_by"] = rejecterID

	if err := s.db.Save(&application).Error; err != nil {
		return nil, fmt.Errorf("failed to update license application: %w", err)
	}

	// Send notification to applicant
	go s.sendRejectionNotification(&application)

	return &application, nil
}

func (s *LicenseService) RevokeLicense(applicationID uuid.UUID, revokerID uuid.UUID, req *RevokeLicenseRequest) (*models.LicenseApplication, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Find application
	var application models.LicenseApplication
	if err := s.db.Preload("IPAsset").Preload("Applicant").
		First(&application, applicationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("license application not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify revoker is the IP creator or admin
	if application.IPAsset.CreatorID != revokerID {
		var revoker models.User
		if err := s.db.First(&revoker, revokerID).Error; err != nil {
			return nil, errors.New("unauthorized to revoke license")
		}
		if revoker.UserType != models.UserTypeAdmin {
			return nil, errors.New("unauthorized to revoke license")
		}
	}

	// Check if license is approved
	if application.Status != models.ApplicationStatusApproved {
		return nil, errors.New("can only revoke approved licenses")
	}

	// Check if there are active products using this license
	var productCount int64
	if err := s.db.Model(&models.Product{}).
		Where("license_id = ? AND status IN (?, ?)",
			applicationID, models.ProductStatusActive, models.ProductStatusDraft).
		Count(&productCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check active products: %w", err)
	}

	if productCount > 0 {
		return nil, errors.New("cannot revoke license with active products")
	}

	// Update application
	application.Status = models.ApplicationStatusRevoked
	application.IsActive = false

	// Update application data with revocation details
	if application.ApplicationData == nil {
		application.ApplicationData = make(models.JSONB)
	}
	application.ApplicationData["revocation_reason"] = req.Reason
	application.ApplicationData["revocation_message"] = req.Message
	application.ApplicationData["revoked_at"] = time.Now()
	application.ApplicationData["revoked_by"] = revokerID

	if err := s.db.Save(&application).Error; err != nil {
		return nil, fmt.Errorf("failed to update license application: %w", err)
	}

	// Send notification to licensee
	go s.sendRevocationNotification(&application)

	return &application, nil
}

func (s *LicenseService) GetLicenseApplication(id uuid.UUID, userID uuid.UUID) (*models.LicenseApplication, error) {
	var application models.LicenseApplication
	if err := s.db.Preload("IPAsset").Preload("IPAsset.Creator").Preload("Applicant").
		Preload("LicenseTerms").Preload("Approver").
		First(&application, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("license application not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check permissions
	if application.ApplicantID != userID && application.IPAsset.CreatorID != userID {
		// Check if user is admin
		var user models.User
		if err := s.db.First(&user, userID).Error; err != nil {
			return nil, errors.New("unauthorized to view license application")
		}
		if user.UserType != models.UserTypeAdmin {
			return nil, errors.New("unauthorized to view license application")
		}
	}

	return &application, nil
}

func (s *LicenseService) SearchLicenseApplications(params LicenseSearchParams, userID uuid.UUID) ([]models.LicenseApplication, int64, error) {
	query := s.db.Model(&models.LicenseApplication{}).
		Preload("IPAsset").Preload("IPAsset.Creator").Preload("Applicant").Preload("LicenseTerms")

	// Apply user-based filtering
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, 0, errors.New("user not found")
	}

	if user.UserType != models.UserTypeAdmin {
		// Non-admin users can only see their own applications or applications for their IP
		query = query.Where("applicant_id = ? OR ip_asset_id IN (SELECT id FROM ip_assets WHERE creator_id = ?)",
			userID, userID)
	}

	// Apply filters
	if params.ApplicantID != nil {
		query = query.Where("applicant_id = ?", *params.ApplicantID)
	}

	if params.IPAssetID != nil {
		query = query.Where("ip_asset_id = ?", *params.IPAssetID)
	}

	if params.CreatorID != nil {
		query = query.Where("ip_asset_id IN (SELECT id FROM ip_assets WHERE creator_id = ?)", *params.CreatorID)
	}

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if params.LicenseType != nil {
		query = query.Joins("JOIN license_terms ON license_applications.license_terms_id = license_terms.id").
			Where("license_terms.license_type = ?", *params.LicenseType)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count license applications: %w", err)
	}

	// Apply sorting
	allowedSortFields := []string{"created_at", "updated_at", "approved_at", "status"}
	query = utils.ApplySort(query, params.PaginationParams, allowedSortFields)

	// Apply pagination
	query = utils.ApplyPagination(query, params.PaginationParams)

	// Execute query
	var applications []models.LicenseApplication
	if err := query.Find(&applications).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch license applications: %w", err)
	}

	return applications, total, nil
}

func (s *LicenseService) GetUserLicenses(userID uuid.UUID, params utils.PaginationParams) ([]models.LicenseApplication, int64, error) {
	query := s.db.Model(&models.LicenseApplication{}).
		Where("applicant_id = ? AND status = ?", userID, models.ApplicationStatusApproved).
		Preload("IPAsset").Preload("IPAsset.Creator").Preload("LicenseTerms")

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count user licenses: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "approved_at"}
	query = utils.ApplySort(query, params, allowedSortFields)
	query = utils.ApplyPagination(query, params)

	// Execute query
	var licenses []models.LicenseApplication
	if err := query.Find(&licenses).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch user licenses: %w", err)
	}

	return licenses, total, nil
}

func (s *LicenseService) VerifyLicense(licenseID uuid.UUID) (*models.LicenseApplication, error) {
	var license models.LicenseApplication
	if err := s.db.Preload("IPAsset").Preload("LicenseTerms").
		First(&license, licenseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("license not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check if license is active and approved
	if license.Status != models.ApplicationStatusApproved || !license.IsActive {
		return nil, errors.New("license is not active")
	}

	// Check expiration if applicable
	if license.ExpiresAt != nil && time.Now().After(*license.ExpiresAt) {
		return nil, errors.New("license has expired")
	}

	return &license, nil
}

// Notification methods

func (s *LicenseService) sendApplicationNotifications(application *models.LicenseApplication, autoApproved bool) {
	if s.notificationService == nil {
		return
	}

	if autoApproved {
		// Send approval notification
		s.notificationService.SendLicenseApprovedNotification(application)
	} else {
		// Send application notification to IP creator
		s.notificationService.SendLicenseApplicationNotification(application)
	}
}

func (s *LicenseService) sendApprovalNotification(application *models.LicenseApplication) {
	if s.notificationService != nil {
		s.notificationService.SendLicenseApprovedNotification(application)
	}
}

func (s *LicenseService) sendRejectionNotification(application *models.LicenseApplication) {
	if s.notificationService != nil {
		s.notificationService.SendLicenseRejectedNotification(application)
	}
}

func (s *LicenseService) sendRevocationNotification(application *models.LicenseApplication) {
	if s.notificationService != nil {
		s.notificationService.SendLicenseRevokedNotification(application)
	}
}

func (s *LicenseService) GetLicenseStatistics(userID uuid.UUID) (map[string]interface{}, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	stats := make(map[string]interface{})

	if user.UserType == models.UserTypeCreator {
		// Creator statistics
		var creatorStats struct {
			TotalIPAssets        int64 `json:"total_ip_assets"`
			TotalApplications    int64 `json:"total_applications"`
			ApprovedLicenses     int64 `json:"approved_licenses"`
			PendingApplications  int64 `json:"pending_applications"`
			RejectedApplications int64 `json:"rejected_applications"`
		}

		// Count IP assets
		s.db.Model(&models.IPAsset{}).Where("creator_id = ?", userID).Count(&creatorStats.TotalIPAssets)

		// Count license applications for creator's IP assets
		s.db.Model(&models.LicenseApplication{}).
			Where("ip_asset_id IN (SELECT id FROM ip_assets WHERE creator_id = ?)", userID).
			Count(&creatorStats.TotalApplications)

		s.db.Model(&models.LicenseApplication{}).
			Where("ip_asset_id IN (SELECT id FROM ip_assets WHERE creator_id = ?) AND status = ?",
				userID, models.ApplicationStatusApproved).
			Count(&creatorStats.ApprovedLicenses)

		s.db.Model(&models.LicenseApplication{}).
			Where("ip_asset_id IN (SELECT id FROM ip_assets WHERE creator_id = ?) AND status = ?",
				userID, models.ApplicationStatusPending).
			Count(&creatorStats.PendingApplications)

		s.db.Model(&models.LicenseApplication{}).
			Where("ip_asset_id IN (SELECT id FROM ip_assets WHERE creator_id = ?) AND status = ?",
				userID, models.ApplicationStatusRejected).
			Count(&creatorStats.RejectedApplications)

		stats["creator_stats"] = creatorStats
	}

	if user.UserType == models.UserTypeSecondaryCreator || user.UserType == models.UserTypeCreator {
		// Licensee statistics
		var licenseeStats struct {
			TotalApplications    int64 `json:"total_applications"`
			ApprovedLicenses     int64 `json:"approved_licenses"`
			PendingApplications  int64 `json:"pending_applications"`
			RejectedApplications int64 `json:"rejected_applications"`
			ActiveProducts       int64 `json:"active_products"`
		}

		// Count applications made by user
		s.db.Model(&models.LicenseApplication{}).Where("applicant_id = ?", userID).Count(&licenseeStats.TotalApplications)

		s.db.Model(&models.LicenseApplication{}).
			Where("applicant_id = ? AND status = ?", userID, models.ApplicationStatusApproved).
			Count(&licenseeStats.ApprovedLicenses)

		s.db.Model(&models.LicenseApplication{}).
			Where("applicant_id = ? AND status = ?", userID, models.ApplicationStatusPending).
			Count(&licenseeStats.PendingApplications)

		s.db.Model(&models.LicenseApplication{}).
			Where("applicant_id = ? AND status = ?", userID, models.ApplicationStatusRejected).
			Count(&licenseeStats.RejectedApplications)

		// Count active products created with licenses
		s.db.Model(&models.Product{}).
			Where("creator_id = ? AND status = ?", userID, models.ProductStatusActive).
			Count(&licenseeStats.ActiveProducts)

		stats["licensee_stats"] = licenseeStats
	}

	return stats, nil
}
