// internal/handlers/auth.go
package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javajoker/imi-backend/internal/i18n"
	"github.com/javajoker/imi-backend/internal/services"
	"github.com/javajoker/imi-backend/internal/utils"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	lang := utils.GetLangFromContext(c)

	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Register user
	authResponse, err := h.authService.Register(&req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"message":       i18n.T(lang, i18n.KeyAuthRegisterSuccess),
		"user":          authResponse.User,
		"token":         authResponse.AccessToken,
		"refresh_token": authResponse.RefreshToken,
		"token_type":    authResponse.TokenType,
		"expires_in":    authResponse.ExpiresIn,
	})
}

// POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	lang := utils.GetLangFromContext(c)

	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Login user
	authResponse, err := h.authService.Login(&req)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message":       i18n.T(lang, i18n.KeyAuthLoginSuccess),
		"user":          authResponse.User,
		"token":         authResponse.AccessToken,
		"refresh_token": authResponse.RefreshToken,
		"token_type":    authResponse.TokenType,
		"expires_in":    authResponse.ExpiresIn,
	})
}

// POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	lang := utils.GetLangFromContext(c)

	// TODO: In a real implementation, you might want to blacklist the token
	// For now, we'll just return success
	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyAuthLogoutSuccess),
	})
}

// POST /auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	lang := utils.GetLangFromContext(c)

	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
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

	// Refresh token
	authResponse, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"user":          authResponse.User,
		"token":         authResponse.AccessToken,
		"refresh_token": authResponse.RefreshToken,
		"token_type":    authResponse.TokenType,
		"expires_in":    authResponse.ExpiresIn,
	})
}

// POST /auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	lang := utils.GetLangFromContext(c)

	var req services.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Send reset email
	if err := h.authService.ForgotPassword(&req); err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyAuthPasswordReset),
	})
}

// POST /auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	lang := utils.GetLangFromContext(c)

	var req services.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// Reset password
	if err := h.authService.ResetPassword(&req); err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, "auth.password_reset_success"),
	})
}

// GET /auth/verify-email/:token
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	token := c.Param("token")

	if token == "" {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationRequired, "token"), nil)
		return
	}

	// Verify email
	if err := h.authService.VerifyEmail(token); err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyVerificationSuccess),
	})
}

// GET /auth/me
func (h *AuthHandler) GetProfile(c *gin.Context) {
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

	// Get user profile
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		utils.NotFoundResponse(c, i18n.KeyUserNotFound)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"user": user,
	})
}
