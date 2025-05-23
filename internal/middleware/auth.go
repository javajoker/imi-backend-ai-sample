// internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/javajoker/imi-backend/internal/i18n"
	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")
		if lang == "" {
			lang = "en"
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": i18n.T(lang, i18n.KeyAuthRequired),
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": i18n.T(lang, i18n.KeyAuthInvalidToken),
			})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := utils.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": i18n.T(lang, i18n.KeyAuthTokenExpired),
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_type", claims.UserType)
		c.Set("verification_level", claims.VerificationLevel)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")
		if lang == "" {
			lang = "en"
		}

		userType, exists := c.Get("user_type")
		if !exists || userType != string(models.UserTypeAdmin) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": i18n.T(lang, i18n.KeyAdminAccessDenied),
			})
			c.Abort()
			return
		}
		c.Next()
	})
}

func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]
		claims, err := utils.ValidateJWT(token)
		if err != nil {
			c.Next()
			return
		}

		// Set user info in context if token is valid
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_type", claims.UserType)
		c.Set("verification_level", claims.VerificationLevel)
		c.Next()
	}
}
