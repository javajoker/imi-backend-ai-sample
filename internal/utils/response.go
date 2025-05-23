// internal/utils/response.go
package utils

import (
	"net/http"

	"github.com/javajoker/imi-backend/internal/i18n"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

func SuccessResponseWithMeta(c *gin.Context, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, code, message string, details interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func BadRequestResponse(c *gin.Context, message string, details interface{}) {
	lang := GetLangFromContext(c)
	if message == "" {
		message = i18n.T(lang, i18n.KeyValidationInvalid, "request")
	}
	ErrorResponse(c, http.StatusBadRequest, "BAD_REQUEST", message, details)
}

func UnauthorizedResponse(c *gin.Context, message string) {
	lang := GetLangFromContext(c)
	if message == "" {
		message = i18n.T(lang, i18n.KeyAuthRequired)
	}
	ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

func ForbiddenResponse(c *gin.Context, message string) {
	lang := GetLangFromContext(c)
	if message == "" {
		message = i18n.T(lang, i18n.KeyAdminAccessDenied)
	}
	ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", message, nil)
}

func NotFoundResponse(c *gin.Context, resource string) {
	lang := GetLangFromContext(c)
	message := i18n.T(lang, resource+".not_found")
	ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", message, nil)
}

func ConflictResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusConflict, "CONFLICT", message, nil)
}

func InternalErrorResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Internal server error"
	}
	ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, nil)
}

func ValidationErrorResponse(c *gin.Context, errors []ValidationError) {
	lang := GetLangFromContext(c)
	message := i18n.T(lang, i18n.KeyValidationInvalid, "input")
	ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", message, errors)
}

func PaginatedResponse(c *gin.Context, result PaginationResult) {
	SetPaginationHeaders(c, result)
	SuccessResponseWithMeta(c, result.Data, gin.H{
		"pagination": gin.H{
			"page":        result.Page,
			"limit":       result.Limit,
			"total":       result.Total,
			"total_pages": result.TotalPages,
		},
	})
}

func GetLangFromContext(c *gin.Context) string {
	if lang, exists := c.Get("lang"); exists {
		if langStr, ok := lang.(string); ok {
			return langStr
		}
	}
	return "en"
}

func GetUserIDFromContext(c *gin.Context) (string, bool) {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr, true
		}
	}
	return "", false
}

func GetUserTypeFromContext(c *gin.Context) (string, bool) {
	if userType, exists := c.Get("user_type"); exists {
		if userTypeStr, ok := userType.(string); ok {
			return userTypeStr, true
		}
	}
	return "", false
}
