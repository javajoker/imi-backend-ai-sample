// internal/services/notification_service.go
package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/config"
	"github.com/javajoker/imi-backend/internal/models"
)

type NotificationService struct {
	db     *gorm.DB
	config *config.Config
}

type EmailTemplate struct {
	Subject string
	Body    string
}

type NotificationRequest struct {
	UserID    uuid.UUID              `json:"user_id" validate:"required"`
	Type      string                 `json:"type" validate:"required"`
	Title     string                 `json:"title" validate:"required"`
	Message   string                 `json:"message" validate:"required"`
	Data      map[string]interface{} `json:"data,omitempty"`
	SendEmail bool                   `json:"send_email,omitempty"`
}

func NewNotificationService(db *gorm.DB, config *config.Config) *NotificationService {
	return &NotificationService{
		db:     db,
		config: config,
	}
}

// Authentication notifications
func (s *NotificationService) SendWelcomeEmail(user *models.User, verificationToken string) error {
	template := s.getEmailTemplate("welcome")

	data := map[string]interface{}{
		"Username":        user.Username,
		"VerificationURL": fmt.Sprintf("%s/verify-email?token=%s", s.config.Frontend.BaseURL, verificationToken),
		"PlatformName":    "IP Marketplace",
	}

	subject := "Welcome to IP Marketplace"
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(user.Email, subject, body)
}

func (s *NotificationService) SendPasswordResetEmail(user *models.User, resetToken string) error {
	template := s.getEmailTemplate("password_reset")

	data := map[string]interface{}{
		"Username":  user.Username,
		"ResetURL":  fmt.Sprintf("%s/reset-password?token=%s", s.config.Frontend.BaseURL, resetToken),
		"ExpiresIn": "1 hour",
	}

	subject := "Password Reset Request"
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(user.Email, subject, body)
}

// License notifications
func (s *NotificationService) SendLicenseApplicationNotification(application *models.LicenseApplication) error {
	// Notify IP creator about new license application
	creator := application.IPAsset.Creator

	notification := &models.AdminNotification{
		Type:                "license_application",
		Title:               "New License Application",
		Message:             fmt.Sprintf("User %s applied for license on your IP asset '%s'", application.Applicant.Username, application.IPAsset.Title),
		Priority:            "medium",
		RelatedResourceType: "license_application",
		RelatedResourceID:   &application.ID,
	}

	if err := s.db.Create(notification).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Send email to creator
	data := map[string]interface{}{
		"CreatorName":    creator.Username,
		"ApplicantName":  application.Applicant.Username,
		"IPAssetTitle":   application.IPAsset.Title,
		"ApplicationURL": fmt.Sprintf("%s/admin/licenses/%s", s.config.Frontend.BaseURL, application.ID),
	}

	subject := "New License Application - " + application.IPAsset.Title
	template := s.getEmailTemplate("license_application")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(creator.Email, subject, body)
}

func (s *NotificationService) SendLicenseApprovedNotification(application *models.LicenseApplication) error {
	applicant := application.Applicant

	data := map[string]interface{}{
		"ApplicantName": applicant.Username,
		"IPAssetTitle":  application.IPAsset.Title,
		"LicenseURL":    fmt.Sprintf("%s/licenses/%s", s.config.Frontend.BaseURL, application.ID),
	}

	subject := "License Approved - " + application.IPAsset.Title
	template := s.getEmailTemplate("license_approved")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(applicant.Email, subject, body)
}

func (s *NotificationService) SendLicenseRejectedNotification(application *models.LicenseApplication) error {
	applicant := application.Applicant

	data := map[string]interface{}{
		"ApplicantName": applicant.Username,
		"IPAssetTitle":  application.IPAsset.Title,
		"Reason":        application.RejectionReason,
	}

	subject := "License Application Rejected - " + application.IPAsset.Title
	template := s.getEmailTemplate("license_rejected")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(applicant.Email, subject, body)
}

func (s *NotificationService) SendLicenseRevokedNotification(application *models.LicenseApplication) error {
	applicant := application.Applicant

	data := map[string]interface{}{
		"ApplicantName": applicant.Username,
		"IPAssetTitle":  application.IPAsset.Title,
		"Reason":        application.ApplicationData["revocation_reason"],
	}

	subject := "License Revoked - " + application.IPAsset.Title
	template := s.getEmailTemplate("license_revoked")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(applicant.Email, subject, body)
}

// Product notifications
func (s *NotificationService) SendPurchaseConfirmationNotification(transaction *models.Transaction) error {
	buyer := transaction.Buyer

	data := map[string]interface{}{
		"BuyerName":       buyer.Username,
		"ProductTitle":    transaction.Product.Title,
		"Amount":          transaction.Amount,
		"TransactionID":   transaction.ID,
		"OrderDetailsURL": fmt.Sprintf("%s/orders/%s", s.config.Frontend.BaseURL, transaction.ID),
	}

	subject := "Purchase Confirmation - " + transaction.Product.Title
	template := s.getEmailTemplate("purchase_confirmation")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(buyer.Email, subject, body)
}

func (s *NotificationService) SendSaleNotification(transaction *models.Transaction) error {
	seller := transaction.Seller

	data := map[string]interface{}{
		"SellerName":    seller.Username,
		"ProductTitle":  transaction.Product.Title,
		"Amount":        transaction.Amount,
		"BuyerName":     transaction.Buyer.Username,
		"TransactionID": transaction.ID,
	}

	subject := "Sale Notification - " + transaction.Product.Title
	template := s.getEmailTemplate("sale_notification")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(seller.Email, subject, body)
}

// Admin notifications
func (s *NotificationService) SendUserStatusChangeNotification(user *models.User, oldStatus models.UserStatus, reason string) error {
	data := map[string]interface{}{
		"Username":  user.Username,
		"NewStatus": user.Status,
		"OldStatus": oldStatus,
		"Reason":    reason,
	}

	subject := "Account Status Update"
	template := s.getEmailTemplate("user_status_change")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(user.Email, subject, body)
}

func (s *NotificationService) SendVerificationUpdateNotification(user *models.User, level models.VerificationLevel) error {
	data := map[string]interface{}{
		"Username":          user.Username,
		"VerificationLevel": level,
	}

	subject := "Verification Status Update"
	template := s.getEmailTemplate("verification_update")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(user.Email, subject, body)
}

func (s *NotificationService) SendIPApprovalNotification(ipAsset *models.IPAsset, message string) error {
	creator := ipAsset.Creator

	data := map[string]interface{}{
		"CreatorName":  creator.Username,
		"IPAssetTitle": ipAsset.Title,
		"Message":      message,
		"IPAssetURL":   fmt.Sprintf("%s/ip-assets/%s", s.config.Frontend.BaseURL, ipAsset.ID),
	}

	subject := "IP Asset Approved - " + ipAsset.Title
	template := s.getEmailTemplate("ip_approval")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(creator.Email, subject, body)
}

func (s *NotificationService) SendIPRejectionNotification(ipAsset *models.IPAsset, reason, message string) error {
	creator := ipAsset.Creator

	data := map[string]interface{}{
		"CreatorName":  creator.Username,
		"IPAssetTitle": ipAsset.Title,
		"Reason":       reason,
		"Message":      message,
	}

	subject := "IP Asset Rejected - " + ipAsset.Title
	template := s.getEmailTemplate("ip_rejection")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(creator.Email, subject, body)
}

func (s *NotificationService) SendRefundNotification(transaction *models.Transaction, reason string) error {
	buyer := transaction.Buyer

	data := map[string]interface{}{
		"BuyerName":     buyer.Username,
		"ProductTitle":  transaction.Product.Title,
		"Amount":        transaction.Amount,
		"Reason":        reason,
		"TransactionID": transaction.ID,
	}

	subject := "Refund Processed - " + transaction.Product.Title
	template := s.getEmailTemplate("refund_notification")
	body, err := s.renderTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.sendEmail(buyer.Email, subject, body)
}

// Generic notification methods
func (s *NotificationService) SendCustomNotification(req *NotificationRequest) error {
	// Create in-app notification
	notification := &models.AdminNotification{
		Type:    req.Type,
		Title:   req.Title,
		Message: req.Message,
	}

	if err := s.db.Create(notification).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Send email if requested
	if req.SendEmail {
		var user models.User
		if err := s.db.First(&user, req.UserID).Error; err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		return s.sendEmail(user.Email, req.Title, req.Message)
	}

	return nil
}

// Helper methods
func (s *NotificationService) sendEmail(to, subject, body string) error {
	if s.config.Email.SMTPHost == "" {
		// Email not configured, just log
		fmt.Printf("Email would be sent to %s: %s\n", to, subject)
		return nil
	}

	// Setup authentication
	auth := smtp.PlainAuth("", s.config.Email.SMTPUsername, s.config.Email.SMTPPassword, s.config.Email.SMTPHost)

	// Compose message
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s", to, subject, body))

	// Send email
	addr := fmt.Sprintf("%s:%s", s.config.Email.SMTPHost, s.config.Email.SMTPPort)
	return smtp.SendMail(addr, auth, s.config.Email.FromEmail, []string{to}, msg)
}

func (s *NotificationService) renderTemplate(templateStr string, data interface{}) (string, error) {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *NotificationService) getEmailTemplate(templateType string) EmailTemplate {
	// In a real implementation, these would be loaded from files or database
	templates := map[string]EmailTemplate{
		"welcome": {
			Subject: "Welcome to IP Marketplace",
			Body: `
<!DOCTYPE html>
<html>
<body>
	<h2>Welcome {{.Username}}!</h2>
	<p>Thank you for joining IP Marketplace. Please verify your email address by clicking the link below:</p>
	<a href="{{.VerificationURL}}">Verify Email</a>
	<p>Best regards,<br>{{.PlatformName}} Team</p>
</body>
</html>`,
		},
		"license_approved": {
			Subject: "License Approved",
			Body: `
<!DOCTYPE html>
<html>
<body>
	<h2>License Approved!</h2>
	<p>Hello {{.ApplicantName}},</p>
	<p>Your license application for "{{.IPAssetTitle}}" has been approved!</p>
	<p>You can now start creating products using this IP.</p>
	<a href="{{.LicenseURL}}">View License Details</a>
	<p>Best regards,<br>IP Marketplace Team</p>
</body>
</html>`,
		},
		// Add more templates as needed...
	}

	if template, exists := templates[templateType]; exists {
		return template
	}

	// Default template
	return EmailTemplate{
		Subject: "Notification",
		Body:    "<p>{{.Message}}</p>",
	}
}
