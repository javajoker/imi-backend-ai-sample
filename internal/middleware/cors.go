// internal/middleware/cors.go
package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://ipmarketplace.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept-Language", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-Total-Count", "X-Page", "X-Per-Page", "X-Total-Pages"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
