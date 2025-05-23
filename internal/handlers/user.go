// internal/handlers/user.go
package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javajoker/imi-backend/internal/i18n"
	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/services"
	"github.com/javajoker/imi-backend/internal/utils"
)

type UserHandler struct {
	userService *services.UserService
}

type UpdateProfileRequest struct {
	Username    string                 `json:"username,omitempty" validate:"omitempty,username"`
	ProfileData map[string]interface{} `json:"profile_data,omitempty"`
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// PUT /users/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	_, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyValidationInvalid, "input"), err.Error())
		return
	}

	// Validate request
	if validationErrors := utils.GetValidationErrors(utils.ValidateStruct(&req)); len(validationErrors) > 0 {
		utils.ValidationErrorResponse(c, validationErrors)
		return
	}

	// TODO: Update profile (would call user service)
	// For now, just return success
	utils.SuccessResponse(c, gin.H{
		"message": i18n.T(lang, i18n.KeyUserProfileUpdated),
	})
}

// GET /users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	// TODO: Get user (would call user service)
	// For now, return placeholder
	utils.SuccessResponse(c, gin.H{
		"user": gin.H{
			"id":       userID,
			"username": "placeholder",
		},
	})
}

// GET /users/:id/public
func (h *UserHandler) GetPublicProfile(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	// TODO: Get public profile (would call user service)
	// For now, return placeholder
	utils.SuccessResponse(c, gin.H{
		"user": gin.H{
			"id":                 userID,
			"username":           "placeholder",
			"verification_level": models.VerificationLevelVerified,
			"created_at":         "2023-01-01T00:00:00Z",
		},
	})
}

// POST /users/upload-avatar
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
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

	// Get uploaded file
	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyFileUploadFailed), err.Error())
		return
	}
	defer file.Close()

	// Validate file size (2MB max for avatars)
	if header.Size > 2*1024*1024 {
		utils.BadRequestResponse(c, i18n.T(lang, i18n.KeyFileTooLarge, "2"), nil)
		return
	}

	// For now, just return success
	utils.SuccessResponse(c, gin.H{
		"message":    i18n.T(lang, i18n.KeyFileUploadSuccess),
		"avatar_url": "https://example.com/avatars/" + userID.String() + ".jpg",
	})
}

// DELETE /users/account
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	lang := utils.GetLangFromContext(c)
	userIDStr, exists := utils.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedResponse(c, "")
		return
	}

	_, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", nil)
		return
	}

	var req struct {
		Password string `json:"password" validate:"required"`
		Reason   string `json:"reason,omitempty"`
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

	// TODO: Delete account (would call user service)
	// For now, just return success
	utils.SuccessResponse(c, gin.H{
		"message": "Account deletion requested successfully",
	})
}
