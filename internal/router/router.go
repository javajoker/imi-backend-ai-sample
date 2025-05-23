// internal/router/router.go
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/config"
	"github.com/javajoker/imi-backend/internal/handlers"
	"github.com/javajoker/imi-backend/internal/middleware"
	"github.com/javajoker/imi-backend/internal/services"
	"github.com/javajoker/imi-backend/internal/utils"
)

func Initialize(db *gorm.DB, cfg *config.Config) *gin.Engine {
	// Initialize services
	notificationService := services.NewNotificationService(db, cfg)
	storageService, _ := services.NewStorageService(cfg)
	blockchainService := services.NewBlockchainService(db, cfg)
	authorizationService := services.NewAuthorizationService(db, blockchainService)

	authService := services.NewAuthService(db, cfg)
	ipService := services.NewIPService(db, blockchainService, storageService)
	licenseService := services.NewLicenseService(db, notificationService)
	productService := services.NewProductService(db, authorizationService, notificationService)
	paymentService := services.NewPaymentService(db, cfg)
	adminService := services.NewAdminService(db, notificationService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(nil) // UserService would be implemented
	ipAssetHandler := handlers.NewIPAssetHandler(ipService, storageService)
	licenseHandler := handlers.NewLicenseHandler(licenseService)
	productHandler := handlers.NewProductHandler(productService, storageService)
	paymentHandler := handlers.NewPaymentHandler(paymentService)
	verificationHandler := handlers.NewVerificationHandler(authorizationService)
	adminHandler := handlers.NewAdminHandler(adminService)

	// Set JWT secret
	utils.SetJWTSecret(cfg.JWT.SecretKey)

	// Initialize Gin router
	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CORS())
	r.Use(middleware.I18nMiddleware())
	r.Use(middleware.GeneralRateLimit())
	r.Use(middleware.AuditLogMiddleware(db))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// API v1 routes
	v1 := r.Group("/v1")
	{
		// Authentication routes
		auth := v1.Group("/auth")
		auth.Use(middleware.AuthRateLimit())
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", middleware.AuthRequired(), authHandler.Logout)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.GET("/verify-email/:token", authHandler.VerifyEmail)
			auth.GET("/me", middleware.AuthRequired(), authHandler.GetProfile)
		}

		// User routes
		users := v1.Group("/users")
		{
			users.GET("/:id", middleware.OptionalAuth(), userHandler.GetUser)
			users.GET("/:id/public", userHandler.GetPublicProfile)

			// Authenticated user routes
			protected := users.Group("")
			protected.Use(middleware.AuthRequired())
			{
				protected.PUT("/profile", userHandler.UpdateProfile)
				protected.POST("/upload-avatar", middleware.UploadRateLimit(), userHandler.UploadAvatar)
				protected.DELETE("/account", userHandler.DeleteAccount)
			}
		}

		// IP Assets routes
		ipAssets := v1.Group("/ip-assets")
		{
			ipAssets.GET("", middleware.OptionalAuth(), ipAssetHandler.GetIPAssets)
			ipAssets.GET("/popular", ipAssetHandler.GetPopularIPAssets)
			ipAssets.GET("/featured", ipAssetHandler.GetFeaturedIPAssets)
			ipAssets.GET("/:id", middleware.OptionalAuth(), ipAssetHandler.GetIPAsset)
			ipAssets.GET("/:id/licenses", ipAssetHandler.GetIPAssetLicenses)

			// Authenticated routes
			protected := ipAssets.Group("")
			protected.Use(middleware.AuthRequired())
			{
				protected.POST("", ipAssetHandler.CreateIPAsset)
				protected.PUT("/:id", ipAssetHandler.UpdateIPAsset)
				protected.DELETE("/:id", ipAssetHandler.DeleteIPAsset)
				protected.POST("/:id/licenses", ipAssetHandler.CreateLicenseTerms)
				protected.GET("/:id/statistics", ipAssetHandler.GetIPAssetStatistics)
				protected.POST("/upload", middleware.UploadRateLimit(), ipAssetHandler.UploadFiles)
			}
		}

		// License routes
		licenses := v1.Group("/licenses")
		licenses.Use(middleware.AuthRequired())
		{
			licenses.POST("/apply", licenseHandler.ApplyForLicense)
			licenses.GET("/applications", licenseHandler.GetLicenseApplications)
			licenses.GET("/my-licenses", licenseHandler.GetMyLicenses)
			licenses.GET("/statistics", licenseHandler.GetLicenseStatistics)
			licenses.GET("/:id", licenseHandler.GetLicenseApplication)
			licenses.PUT("/:id/approve", licenseHandler.ApproveLicense)
			licenses.PUT("/:id/reject", licenseHandler.RejectLicense)
			licenses.PUT("/:id/revoke", licenseHandler.RevokeLicense)
			licenses.GET("/:id/verify", licenseHandler.VerifyLicense)
		}

		// Product routes
		products := v1.Group("/products")
		{
			products.GET("", middleware.OptionalAuth(), productHandler.GetProducts)
			products.GET("/popular", productHandler.GetPopularProducts)
			products.GET("/featured", productHandler.GetFeaturedProducts)
			products.GET("/:id", middleware.OptionalAuth(), productHandler.GetProduct)
			products.GET("/:id/verify", productHandler.VerifyProduct)

			// Authenticated routes
			protected := products.Group("")
			protected.Use(middleware.AuthRequired())
			{
				protected.POST("", productHandler.CreateProduct)
				protected.PUT("/:id", productHandler.UpdateProduct)
				protected.DELETE("/:id", productHandler.DeleteProduct)
				protected.POST("/:id/purchase", productHandler.PurchaseProduct)
				protected.GET("/:id/statistics", productHandler.GetProductStatistics)
				protected.POST("/upload-images", middleware.UploadRateLimit(), productHandler.UploadProductImages)
			}
		}

		// Payment routes
		payments := v1.Group("/payments")
		payments.Use(middleware.AuthRequired())
		{
			payments.POST("/intent", paymentHandler.CreatePaymentIntent)
			payments.POST("/confirm", paymentHandler.ConfirmPayment)
			payments.GET("/history", paymentHandler.GetPaymentHistory)
			payments.GET("/balance", paymentHandler.GetUserBalance)
			payments.POST("/payout", paymentHandler.RequestPayout)
			payments.POST("/refund", middleware.AdminRequired(), paymentHandler.ProcessRefund)
		}

		// Verification routes (public)
		verify := v1.Group("/verify")
		{
			verify.GET("/:code", verificationHandler.VerifyProductByCode)
			verify.GET("/chain/:id", verificationHandler.VerifyAuthorizationChain)
			verify.GET("/chain/:id/history", verificationHandler.GetAuthorizationChainHistory)
		}

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired(), middleware.AdminRequired())
		{
			// Dashboard
			dashboard := admin.Group("/dashboard")
			{
				dashboard.GET("/stats", adminHandler.GetDashboardStats)
			}

			// User management
			adminUsers := admin.Group("/users")
			{
				adminUsers.GET("", adminHandler.GetUsers)
				adminUsers.PUT("/:id/status", adminHandler.UpdateUserStatus)
				adminUsers.PUT("/:id/verify", adminHandler.UpdateUserVerification)
			}

			// IP asset management
			adminIP := admin.Group("/ip-assets")
			{
				adminIP.GET("/pending", adminHandler.GetPendingIPAssets)
				adminIP.PUT("/:id/approve", adminHandler.ApproveIPAsset)
				adminIP.PUT("/:id/reject", adminHandler.RejectIPAsset)
			}

			// License management
			adminLicenses := admin.Group("/licenses")
			{
				adminLicenses.GET("", adminHandler.GetLicenseApplications)
			}

			// Transaction management
			adminTransactions := admin.Group("/transactions")
			{
				adminTransactions.GET("", adminHandler.GetTransactions)
				adminTransactions.POST("/:id/refund", adminHandler.ProcessRefund)
			}

			// Analytics and reporting
			adminAnalytics := admin.Group("/analytics")
			{
				adminAnalytics.GET("", adminHandler.GetAnalytics)
			}

			// Settings management
			adminSettings := admin.Group("/settings")
			{
				adminSettings.GET("", adminHandler.GetSettings)
				adminSettings.PUT("", adminHandler.UpdateSettings)
			}

			// Content moderation
			adminReports := admin.Group("/reports")
			{
				adminReports.GET("/content", adminHandler.GetContentReports)
				adminReports.PUT("/:id/resolve", adminHandler.ResolveContentReport)
			}
		}

		// Search routes
		search := v1.Group("/search")
		{
			search.GET("/ip-assets", middleware.OptionalAuth(), ipAssetHandler.GetIPAssets)
			search.GET("/products", middleware.OptionalAuth(), productHandler.GetProducts)
		}

		// Category routes
		categories := v1.Group("/categories")
		{
			categories.GET("", getCategoriesHandler)
			categories.GET("/:category/ip-assets", middleware.OptionalAuth(), ipAssetHandler.GetIPAssets)
			categories.GET("/:category/products", middleware.OptionalAuth(), productHandler.GetProducts)
		}

		// Statistics routes (public)
		stats := v1.Group("/stats")
		{
			stats.GET("/platform", getPlatformStatsHandler)
		}
	}

	// Static file serving (for development)
	if cfg.Environment == "development" {
		r.Static("/uploads", "./uploads")
	}

	return r
}

// Helper handlers for simple endpoints
func getCategoriesHandler(c *gin.Context) {
	categories := []map[string]interface{}{
		{"id": "art", "name": "Art & Design", "icon": "palette"},
		{"id": "gaming", "name": "Gaming", "icon": "gamepad"},
		{"id": "music", "name": "Music & Audio", "icon": "music"},
		{"id": "video", "name": "Video & Animation", "icon": "video"},
		{"id": "photography", "name": "Photography", "icon": "camera"},
		{"id": "3d", "name": "3D Models", "icon": "cube"},
		{"id": "software", "name": "Software", "icon": "code"},
		{"id": "fashion", "name": "Fashion", "icon": "shirt"},
		{"id": "education", "name": "Education", "icon": "book"},
		{"id": "business", "name": "Business", "icon": "briefcase"},
	}

	utils.SuccessResponse(c, gin.H{
		"categories": categories,
	})
}

func getPlatformStatsHandler(c *gin.Context) {
	// TODO: This would typically come from a service
	// For now, return mock data
	stats := map[string]interface{}{
		"total_users":     15420,
		"total_ip_assets": 3240,
		"total_products":  8960,
		"total_licenses":  5670,
		"total_revenue":   285000.50,
		"active_creators": 2890,
		"verified_assets": 2980,
		"last_updated":    "2024-01-15T10:30:00Z",
	}

	utils.SuccessResponse(c, gin.H{
		"stats": stats,
	})
}
