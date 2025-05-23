// internal/services/admin_service.go
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

type AdminService struct {
	db                  *gorm.DB
	notificationService *NotificationService
}

type AdminDashboardStats struct {
	TotalUsers            int64   `json:"total_users"`
	ActiveUsers           int64   `json:"active_users"`
	NewUsersThisMonth     int64   `json:"new_users_this_month"`
	TotalRevenue          float64 `json:"total_revenue"`
	MonthlyRevenue        float64 `json:"monthly_revenue"`
	TotalIPs              int64   `json:"total_ips"`
	PendingIPVerification int64   `json:"pending_ip_verification"`
	TotalProducts         int64   `json:"total_products"`
	ActiveLicenses        int64   `json:"active_licenses"`
	PendingLicenses       int64   `json:"pending_licenses"`
	TotalTransactions     int64   `json:"total_transactions"`
	UserGrowth            float64 `json:"user_growth"`
	RevenueGrowth         float64 `json:"revenue_growth"`
}

type AdminUserFilter struct {
	utils.PaginationParams
	UserType          *models.UserType          `json:"user_type,omitempty"`
	Status            *models.UserStatus        `json:"status,omitempty"`
	VerificationLevel *models.VerificationLevel `json:"verification_level,omitempty"`
	CreatedAfter      *time.Time                `json:"created_after,omitempty"`
	CreatedBefore     *time.Time                `json:"created_before,omitempty"`
}

type AdminIPFilter struct {
	utils.PaginationParams
	CreatorID          *uuid.UUID                 `json:"creator_id,omitempty"`
	VerificationStatus *models.VerificationStatus `json:"verification_status,omitempty"`
	Status             *models.ProductStatus      `json:"status,omitempty"`
	CreatedAfter       *time.Time                 `json:"created_after,omitempty"`
	CreatedBefore      *time.Time                 `json:"created_before,omitempty"`
}

type AdminLicenseFilter struct {
	utils.PaginationParams
	ApplicantID   *uuid.UUID                `json:"applicant_id,omitempty"`
	IPAssetID     *uuid.UUID                `json:"ip_asset_id,omitempty"`
	Status        *models.ApplicationStatus `json:"status,omitempty"`
	LicenseType   *models.LicenseType       `json:"license_type,omitempty"`
	CreatedAfter  *time.Time                `json:"created_after,omitempty"`
	CreatedBefore *time.Time                `json:"created_before,omitempty"`
}

type AdminTransactionFilter struct {
	utils.PaginationParams
	TransactionType *models.TransactionType   `json:"transaction_type,omitempty"`
	Status          *models.TransactionStatus `json:"status,omitempty"`
	BuyerID         *uuid.UUID                `json:"buyer_id,omitempty"`
	SellerID        *uuid.UUID                `json:"seller_id,omitempty"`
	AmountMin       *float64                  `json:"amount_min,omitempty"`
	AmountMax       *float64                  `json:"amount_max,omitempty"`
	CreatedAfter    *time.Time                `json:"created_after,omitempty"`
	CreatedBefore   *time.Time                `json:"created_before,omitempty"`
}

func NewAdminService(db *gorm.DB, notificationService *NotificationService) *AdminService {
	return &AdminService{
		db:                  db,
		notificationService: notificationService,
	}
}

// Dashboard Statistics
func (s *AdminService) GetDashboardStats() (*AdminDashboardStats, error) {
	stats := &AdminDashboardStats{}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := monthStart.AddDate(0, -1, 0)

	// User statistics
	s.db.Model(&models.User{}).Count(&stats.TotalUsers)
	s.db.Model(&models.User{}).Where("status = ?", models.UserStatusActive).Count(&stats.ActiveUsers)
	s.db.Model(&models.User{}).Where("created_at >= ?", monthStart).Count(&stats.NewUsersThisMonth)

	// Revenue statistics
	s.db.Model(&models.Transaction{}).
		Where("status = ?", models.TransactionStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalRevenue)

	s.db.Model(&models.Transaction{}).
		Where("status = ? AND created_at >= ?", models.TransactionStatusCompleted, monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.MonthlyRevenue)

	// IP and Product statistics
	s.db.Model(&models.IPAsset{}).Where("status = ?", models.ProductStatusActive).Count(&stats.TotalIPs)
	s.db.Model(&models.IPAsset{}).
		Where("verification_status = ?", models.VerificationStatusPending).
		Count(&stats.PendingIPVerification)

	s.db.Model(&models.Product{}).Where("status = ?", models.ProductStatusActive).Count(&stats.TotalProducts)

	// License statistics
	s.db.Model(&models.LicenseApplication{}).
		Where("status = ?", models.ApplicationStatusApproved).Count(&stats.ActiveLicenses)
	s.db.Model(&models.LicenseApplication{}).
		Where("status = ?", models.ApplicationStatusPending).Count(&stats.PendingLicenses)

	// Transaction statistics
	s.db.Model(&models.Transaction{}).Count(&stats.TotalTransactions)

	// Growth calculations
	var lastMonthUsers int64
	// var lastMonthRevenue int64
	s.db.Model(&models.User{}).
		Where("created_at >= ? AND created_at < ?", lastMonthStart, monthStart).
		Count(&lastMonthUsers)

	var lastMonthRevenueAmount float64
	s.db.Model(&models.Transaction{}).
		Where("status = ? AND created_at >= ? AND created_at < ?",
			models.TransactionStatusCompleted, lastMonthStart, monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&lastMonthRevenueAmount)

	if lastMonthUsers > 0 {
		stats.UserGrowth = float64(stats.NewUsersThisMonth-lastMonthUsers) / float64(lastMonthUsers) * 100
	}

	if lastMonthRevenueAmount > 0 {
		stats.RevenueGrowth = (stats.MonthlyRevenue - lastMonthRevenueAmount) / lastMonthRevenueAmount * 100
	}

	return stats, nil
}

// User Management
func (s *AdminService) GetUsers(filter AdminUserFilter) ([]models.User, int64, error) {
	query := s.db.Model(&models.User{})

	// Apply filters
	if filter.UserType != nil {
		query = query.Where("user_type = ?", *filter.UserType)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.VerificationLevel != nil {
		query = query.Where("verification_level = ?", *filter.VerificationLevel)
	}
	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("username ILIKE ? OR email ILIKE ?", searchTerm, searchTerm)
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filter.CreatedBefore)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "updated_at", "username", "email", "user_type", "status"}
	query = utils.ApplySort(query, filter.PaginationParams, allowedSortFields)
	query = utils.ApplyPagination(query, filter.PaginationParams)

	// Execute query
	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch users: %w", err)
	}

	return users, total, nil
}

func (s *AdminService) UpdateUserStatus(userID uuid.UUID, status models.UserStatus, adminID uuid.UUID, reason string) error {
	// Find user
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	// Prevent admins from modifying other admins
	if user.UserType == models.UserTypeAdmin {
		var admin models.User
		if err := s.db.First(&admin, adminID).Error; err != nil {
			return errors.New("admin not found")
		}
		if admin.UserType != models.UserTypeAdmin || admin.ID != user.ID {
			return errors.New("cannot modify admin user status")
		}
	}

	oldStatus := user.Status
	user.Status = status

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	// Create audit log
	go s.createAuditLog(adminID, "UPDATE_USER_STATUS", "user", &userID,
		map[string]interface{}{"status": oldStatus},
		map[string]interface{}{"status": status, "reason": reason})

	// Send notification to user
	go s.sendUserStatusNotification(&user, oldStatus, reason)

	return nil
}

func (s *AdminService) UpdateUserVerificationLevel(userID uuid.UUID, level models.VerificationLevel, adminID uuid.UUID) error {
	// Find user
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	oldLevel := user.VerificationLevel
	user.VerificationLevel = level

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update verification level: %w", err)
	}

	// Create audit log
	go s.createAuditLog(adminID, "UPDATE_USER_VERIFICATION", "user", &userID,
		map[string]interface{}{"verification_level": oldLevel},
		map[string]interface{}{"verification_level": level})

	// Send notification to user
	go s.sendVerificationUpdateNotification(&user, level)

	return nil
}

// IP Asset Management
func (s *AdminService) GetPendingIPAssets(filter AdminIPFilter) ([]models.IPAsset, int64, error) {
	query := s.db.Model(&models.IPAsset{}).Preload("Creator")

	// Apply filters
	if filter.VerificationStatus != nil {
		query = query.Where("verification_status = ?", *filter.VerificationStatus)
	} else {
		query = query.Where("verification_status = ?", models.VerificationStatusPending)
	}

	if filter.CreatorID != nil {
		query = query.Where("creator_id = ?", *filter.CreatorID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filter.CreatedBefore)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count IP assets: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "updated_at", "title", "verification_status"}
	query = utils.ApplySort(query, filter.PaginationParams, allowedSortFields)
	query = utils.ApplyPagination(query, filter.PaginationParams)

	// Execute query
	var ipAssets []models.IPAsset
	if err := query.Find(&ipAssets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch IP assets: %w", err)
	}

	return ipAssets, total, nil
}

func (s *AdminService) ApproveIPAsset(ipAssetID uuid.UUID, adminID uuid.UUID, message string) error {
	// Find IP asset
	var ipAsset models.IPAsset
	if err := s.db.Preload("Creator").First(&ipAsset, ipAssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("IP asset not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if ipAsset.VerificationStatus != models.VerificationStatusPending {
		return errors.New("IP asset is not pending verification")
	}

	// Update status
	ipAsset.VerificationStatus = models.VerificationStatusApproved
	if ipAsset.Metadata == nil {
		ipAsset.Metadata = make(models.JSONB)
	}
	ipAsset.Metadata["approved_by"] = adminID
	ipAsset.Metadata["approved_at"] = time.Now()
	if message != "" {
		ipAsset.Metadata["approval_message"] = message
	}

	if err := s.db.Save(&ipAsset).Error; err != nil {
		return fmt.Errorf("failed to approve IP asset: %w", err)
	}

	// Create audit log
	go s.createAuditLog(adminID, "APPROVE_IP_ASSET", "ip_asset", &ipAssetID, nil,
		map[string]interface{}{"verification_status": models.VerificationStatusApproved, "message": message})

	// Send notification to creator
	go s.sendIPApprovalNotification(&ipAsset, message)

	return nil
}

func (s *AdminService) RejectIPAsset(ipAssetID uuid.UUID, adminID uuid.UUID, reason, message string) error {
	// Find IP asset
	var ipAsset models.IPAsset
	if err := s.db.Preload("Creator").First(&ipAsset, ipAssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("IP asset not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if ipAsset.VerificationStatus != models.VerificationStatusPending {
		return errors.New("IP asset is not pending verification")
	}

	// Update status
	ipAsset.VerificationStatus = models.VerificationStatusRejected
	if ipAsset.Metadata == nil {
		ipAsset.Metadata = make(models.JSONB)
	}
	ipAsset.Metadata["rejected_by"] = adminID
	ipAsset.Metadata["rejected_at"] = time.Now()
	ipAsset.Metadata["rejection_reason"] = reason
	if message != "" {
		ipAsset.Metadata["rejection_message"] = message
	}

	if err := s.db.Save(&ipAsset).Error; err != nil {
		return fmt.Errorf("failed to reject IP asset: %w", err)
	}

	// Create audit log
	go s.createAuditLog(adminID, "REJECT_IP_ASSET", "ip_asset", &ipAssetID, nil,
		map[string]interface{}{"verification_status": models.VerificationStatusRejected, "reason": reason, "message": message})

	// Send notification to creator
	go s.sendIPRejectionNotification(&ipAsset, reason, message)

	return nil
}

// License Management
func (s *AdminService) GetLicenseApplications(filter AdminLicenseFilter) ([]models.LicenseApplication, int64, error) {
	query := s.db.Model(&models.LicenseApplication{}).
		Preload("IPAsset").Preload("IPAsset.Creator").Preload("Applicant").Preload("LicenseTerms")

	// Apply filters
	if filter.ApplicantID != nil {
		query = query.Where("applicant_id = ?", *filter.ApplicantID)
	}
	if filter.IPAssetID != nil {
		query = query.Where("ip_asset_id = ?", *filter.IPAssetID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.LicenseType != nil {
		query = query.Joins("JOIN license_terms ON license_applications.license_terms_id = license_terms.id").
			Where("license_terms.license_type = ?", *filter.LicenseType)
	}
	if filter.CreatedAfter != nil {
		query = query.Where("license_applications.created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("license_applications.created_at <= ?", *filter.CreatedBefore)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count license applications: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "updated_at", "approved_at", "status"}
	query = utils.ApplySort(query, filter.PaginationParams, allowedSortFields)
	query = utils.ApplyPagination(query, filter.PaginationParams)

	// Execute query
	var applications []models.LicenseApplication
	if err := query.Find(&applications).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch license applications: %w", err)
	}

	return applications, total, nil
}

// Transaction Management
func (s *AdminService) GetTransactions(filter AdminTransactionFilter) ([]models.Transaction, int64, error) {
	query := s.db.Model(&models.Transaction{}).Preload("Buyer").Preload("Seller").Preload("Product")

	// Apply filters
	if filter.TransactionType != nil {
		query = query.Where("transaction_type = ?", *filter.TransactionType)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.BuyerID != nil {
		query = query.Where("buyer_id = ?", *filter.BuyerID)
	}
	if filter.SellerID != nil {
		query = query.Where("seller_id = ?", *filter.SellerID)
	}
	if filter.AmountMin != nil {
		query = query.Where("amount >= ?", *filter.AmountMin)
	}
	if filter.AmountMax != nil {
		query = query.Where("amount <= ?", *filter.AmountMax)
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filter.CreatedBefore)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "updated_at", "amount", "status", "processed_at"}
	query = utils.ApplySort(query, filter.PaginationParams, allowedSortFields)
	query = utils.ApplyPagination(query, filter.PaginationParams)

	// Execute query
	var transactions []models.Transaction
	if err := query.Find(&transactions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	return transactions, total, nil
}

func (s *AdminService) ProcessRefund(transactionID uuid.UUID, adminID uuid.UUID, reason string) error {
	// Find transaction
	var transaction models.Transaction
	if err := s.db.Preload("Buyer").Preload("Seller").Preload("Product").
		First(&transaction, transactionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("transaction not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if transaction.Status != models.TransactionStatusCompleted {
		return errors.New("can only refund completed transactions")
	}

	// Update transaction status
	now := time.Now()
	transaction.Status = models.TransactionStatusRefunded
	transaction.RefundedAt = &now
	transaction.RefundReason = reason

	if err := s.db.Save(&transaction).Error; err != nil {
		return fmt.Errorf("failed to process refund: %w", err)
	}

	// Create audit log
	go s.createAuditLog(adminID, "PROCESS_REFUND", "transaction", &transactionID, nil,
		map[string]interface{}{"status": models.TransactionStatusRefunded, "reason": reason})

	// Send notifications
	go s.sendRefundNotification(&transaction, reason)

	return nil
}

// Settings Management
func (s *AdminService) GetSettings() (map[string]models.AdminSettings, error) {
	var settings []models.AdminSettings
	if err := s.db.Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch settings: %w", err)
	}

	settingsMap := make(map[string]models.AdminSettings)
	for _, setting := range settings {
		key := fmt.Sprintf("%s.%s", setting.Category, setting.Key)
		settingsMap[key] = setting
	}

	return settingsMap, nil
}

func (s *AdminService) UpdateSetting(category, key string, value interface{}, dataType string, adminID uuid.UUID) error {
	var setting models.AdminSettings
	err := s.db.Where("category = ? AND key = ?", category, key).First(&setting).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new setting
		setting = models.AdminSettings{
			Category:  category,
			Key:       key,
			Value:     models.JSONB{"value": value},
			DataType:  dataType,
			UpdatedBy: adminID,
		}
		if err := s.db.Create(&setting).Error; err != nil {
			return fmt.Errorf("failed to create setting: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("database error: %w", err)
	} else {
		// Update existing setting
		oldValue := setting.Value
		setting.Value = models.JSONB{"value": value}
		setting.DataType = dataType
		setting.UpdatedBy = adminID

		if err := s.db.Save(&setting).Error; err != nil {
			return fmt.Errorf("failed to update setting: %w", err)
		}

		// Create audit log
		go s.createAuditLog(adminID, "UPDATE_SETTING", "admin_setting", &setting.ID,
			map[string]interface{}{"value": oldValue},
			map[string]interface{}{"value": setting.Value})
	}

	return nil
}

// Analytics and Reporting
func (s *AdminService) GetAnalytics(startDate, endDate time.Time, metrics []string) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	for _, metric := range metrics {
		switch metric {
		case "user_registrations":
			var count int64
			s.db.Model(&models.User{}).
				Where("created_at BETWEEN ? AND ?", startDate, endDate).
				Count(&count)
			analytics["user_registrations"] = count

		case "ip_creations":
			var count int64
			s.db.Model(&models.IPAsset{}).
				Where("created_at BETWEEN ? AND ?", startDate, endDate).
				Count(&count)
			analytics["ip_creations"] = count

		case "license_applications":
			var count int64
			s.db.Model(&models.LicenseApplication{}).
				Where("created_at BETWEEN ? AND ?", startDate, endDate).
				Count(&count)
			analytics["license_applications"] = count

		case "product_sales":
			var count int64
			s.db.Model(&models.Transaction{}).
				Where("transaction_type = ? AND created_at BETWEEN ? AND ?",
					models.TransactionTypeProductSale, startDate, endDate).
				Count(&count)
			analytics["product_sales"] = count

		case "revenue":
			var revenue float64
			s.db.Model(&models.Transaction{}).
				Where("status = ? AND created_at BETWEEN ? AND ?",
					models.TransactionStatusCompleted, startDate, endDate).
				Select("COALESCE(SUM(amount), 0)").Scan(&revenue)
			analytics["revenue"] = revenue
		}
	}

	return analytics, nil
}

// Helper methods
func (s *AdminService) createAuditLog(userID uuid.UUID, action, resourceType string, resourceID *uuid.UUID, oldValues, newValues map[string]interface{}) {
	auditLog := &models.AuditLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValues:    models.JSONB(oldValues),
		NewValues:    models.JSONB(newValues),
	}

	s.db.Create(auditLog)
}

func (s *AdminService) sendUserStatusNotification(user *models.User, oldStatus models.UserStatus, reason string) {
	if s.notificationService != nil {
		s.notificationService.SendUserStatusChangeNotification(user, oldStatus, reason)
	}
}

func (s *AdminService) sendVerificationUpdateNotification(user *models.User, level models.VerificationLevel) {
	if s.notificationService != nil {
		s.notificationService.SendVerificationUpdateNotification(user, level)
	}
}

func (s *AdminService) sendIPApprovalNotification(ipAsset *models.IPAsset, message string) {
	if s.notificationService != nil {
		s.notificationService.SendIPApprovalNotification(ipAsset, message)
	}
}

func (s *AdminService) sendIPRejectionNotification(ipAsset *models.IPAsset, reason, message string) {
	if s.notificationService != nil {
		s.notificationService.SendIPRejectionNotification(ipAsset, reason, message)
	}
}

func (s *AdminService) sendRefundNotification(transaction *models.Transaction, reason string) {
	if s.notificationService != nil {
		s.notificationService.SendRefundNotification(transaction, reason)
	}
}

// Content Moderation
func (s *AdminService) GetContentReports(params utils.PaginationParams) ([]models.ContentReport, int64, error) {
	query := s.db.Model(&models.ContentReport{}).Preload("Reporter").Preload("Resolver")

	if params.Search != "" {
		query = query.Where("reason ILIKE ? OR description ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count content reports: %w", err)
	}

	// Apply sorting and pagination
	allowedSortFields := []string{"created_at", "updated_at", "status", "resolved_at"}
	query = utils.ApplySort(query, params, allowedSortFields)
	query = utils.ApplyPagination(query, params)

	// Execute query
	var reports []models.ContentReport
	if err := query.Find(&reports).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch content reports: %w", err)
	}

	return reports, total, nil
}

func (s *AdminService) ResolveContentReport(reportID uuid.UUID, adminID uuid.UUID, action, notes string) error {
	var report models.ContentReport
	if err := s.db.First(&report, reportID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("content report not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if report.Status != "pending" {
		return errors.New("report is already resolved")
	}

	// Update report
	now := time.Now()
	report.Status = "resolved"
	report.AdminNotes = notes
	report.ResolvedBy = &adminID
	report.ResolvedAt = &now

	if err := s.db.Save(&report).Error; err != nil {
		return fmt.Errorf("failed to resolve report: %w", err)
	}

	// Create audit log
	go s.createAuditLog(adminID, "RESOLVE_CONTENT_REPORT", "content_report", &reportID, nil,
		map[string]interface{}{"action": action, "notes": notes})

	return nil
}
