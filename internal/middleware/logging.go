// internal/middleware/logging.go
package middleware

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/models"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func AuditLogMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for GET requests and health checks
		if c.Request.Method == "GET" || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = ioutil.ReadAll(c.Request.Body)
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(requestBody))
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Create audit log
		userID, _ := c.Get("user_id")
		var userUUID *uuid.UUID
		if userID != nil {
			if uid, ok := userID.(string); ok {
				if parsed, err := uuid.Parse(uid); err == nil {
					userUUID = &parsed
				}
			}
		}

		// Parse request body for old/new values
		var requestData map[string]interface{}
		if len(requestBody) > 0 {
			json.Unmarshal(requestBody, &requestData)
		}

		auditLog := &models.AuditLog{
			UserID:       userUUID,
			Action:       c.Request.Method + " " + c.Request.URL.Path,
			ResourceType: extractResourceType(c.Request.URL.Path),
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			NewValues:    models.JSONB(requestData),
		}

		// Extract resource ID from URL if present
		if resourceID := extractResourceID(c.Request.URL.Path); resourceID != "" {
			if parsed, err := uuid.Parse(resourceID); err == nil {
				auditLog.ResourceID = &parsed
			}
		}

		// Save audit log asynchronously
		go func() {
			if err := db.Create(auditLog).Error; err != nil {
				logrus.WithError(err).Error("Failed to create audit log")
			}
		}()

		// Log the request
		logrus.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"duration":   duration.Milliseconds(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"user_id":    userID,
		}).Info("Request processed")
	}
}

func extractResourceType(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "v1" {
		return parts[1]
	}
	if len(parts) >= 1 {
		return parts[0]
	}
	return "unknown"
}

func extractResourceID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for _, part := range parts {
		if _, err := uuid.Parse(part); err == nil {
			return part
		}
	}
	return ""
}

func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
	})
}
