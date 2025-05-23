// internal/i18n/keys.go
package i18n

// Translation keys constants
const (
	// Common
	KeySuccess = "success"
	KeyError   = "error"
	KeyWarning = "warning"
	KeyInfo    = "info"

	// Authentication
	KeyAuthRequired           = "auth.required"
	KeyAuthInvalidToken       = "auth.invalid_token"
	KeyAuthTokenExpired       = "auth.token_expired"
	KeyAuthInvalidCredentials = "auth.invalid_credentials"
	KeyAuthUserNotFound       = "auth.user_not_found"
	KeyAuthUserExists         = "auth.user_exists"
	KeyAuthEmailNotVerified   = "auth.email_not_verified"
	KeyAuthLoginSuccess       = "auth.login_success"
	KeyAuthLogoutSuccess      = "auth.logout_success"
	KeyAuthRegisterSuccess    = "auth.register_success"
	KeyAuthPasswordReset      = "auth.password_reset"

	// User Management
	KeyUserProfileUpdated      = "user.profile_updated"
	KeyUserNotFound            = "user.not_found"
	KeyUserSuspended           = "user.suspended"
	KeyUserVerified            = "user.verified"
	KeyUserVerificationPending = "user.verification_pending"

	// IP Assets
	KeyIPAssetCreated             = "ip_asset.created"
	KeyIPAssetUpdated             = "ip_asset.updated"
	KeyIPAssetDeleted             = "ip_asset.deleted"
	KeyIPAssetNotFound            = "ip_asset.not_found"
	KeyIPAssetApproved            = "ip_asset.approved"
	KeyIPAssetRejected            = "ip_asset.rejected"
	KeyIPAssetVerificationPending = "ip_asset.verification_pending"

	// Licenses
	KeyLicenseApplied  = "license.applied"
	KeyLicenseApproved = "license.approved"
	KeyLicenseRejected = "license.rejected"
	KeyLicenseRevoked  = "license.revoked"
	KeyLicenseNotFound = "license.not_found"
	KeyLicenseExpired  = "license.expired"
	KeyLicenseInvalid  = "license.invalid"

	// Products
	KeyProductCreated    = "product.created"
	KeyProductUpdated    = "product.updated"
	KeyProductDeleted    = "product.deleted"
	KeyProductNotFound   = "product.not_found"
	KeyProductPurchased  = "product.purchased"
	KeyProductOutOfStock = "product.out_of_stock"

	// Payments
	KeyPaymentSuccess        = "payment.success"
	KeyPaymentFailed         = "payment.failed"
	KeyPaymentPending        = "payment.pending"
	KeyPaymentRefunded       = "payment.refunded"
	KeyPaymentInvalidAmount  = "payment.invalid_amount"
	KeyPaymentMethodRequired = "payment.method_required"

	// Admin
	KeyAdminActionSuccess   = "admin.action_success"
	KeyAdminAccessDenied    = "admin.access_denied"
	KeyAdminUserSuspended   = "admin.user_suspended"
	KeyAdminUserUnsuspended = "admin.user_unsuspended"
	KeyAdminSettingsUpdated = "admin.settings_updated"

	// Validation
	KeyValidationRequired = "validation.required"
	KeyValidationInvalid  = "validation.invalid"
	KeyValidationTooShort = "validation.too_short"
	KeyValidationTooLong  = "validation.too_long"
	KeyValidationEmail    = "validation.invalid_email"
	KeyValidationPassword = "validation.invalid_password"

	// File Upload
	KeyFileUploadSuccess = "file.upload_success"
	KeyFileUploadFailed  = "file.upload_failed"
	KeyFileInvalidType   = "file.invalid_type"
	KeyFileTooLarge      = "file.too_large"

	// Search
	KeySearchNoResults    = "search.no_results"
	KeySearchResultsFound = "search.results_found"

	// Verification
	KeyVerificationSuccess = "verification.success"
	KeyVerificationFailed  = "verification.failed"
	KeyVerificationInvalid = "verification.invalid_code"

	// Notifications
	KeyNotificationSent   = "notification.sent"
	KeyNotificationFailed = "notification.failed"
)
