// internal/database/connection.go
package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javajoker/imi-backend/internal/config"
	"github.com/javajoker/imi-backend/internal/models"
)

var DB *gorm.DB

func Initialize(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var err error
	var gormConfig *gorm.Config

	// Configure GORM logger
	if cfg.LogLevel == "silent" {
		gormConfig = &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		}
	} else {
		gormConfig = &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		}
	}

	// Connect to database
	DB, err = gorm.Open(postgres.Open(cfg.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established successfully")
	return DB, nil
}

func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting underlying sql.DB: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	} else {
		log.Println("Database connection closed successfully")
	}
}

func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Enable UUID extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create UUID extension: %w", err)
	}

	// Run auto-migrations
	err := db.AutoMigrate(
		&models.User{},
		&models.IPAsset{},
		&models.LicenseTerms{},
		&models.LicenseApplication{},
		&models.Product{},
		&models.Transaction{},
		&models.AuthorizationChain{},
		&models.AdminSettings{},
		&models.AuditLog{},
		&models.AdminNotification{},
		&models.ContentReport{},
		&models.PlatformAnalytics{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create indexes
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

func createIndexes(db *gorm.DB) error {
	indexes := []string{
		// User indexes
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_type_status ON users(user_type, status)",
		"CREATE INDEX IF NOT EXISTS idx_users_verification_level ON users(verification_level)",

		// IP Asset indexes
		"CREATE INDEX IF NOT EXISTS idx_ip_assets_creator ON ip_assets(creator_id)",
		"CREATE INDEX IF NOT EXISTS idx_ip_assets_category ON ip_assets(category)",
		"CREATE INDEX IF NOT EXISTS idx_ip_assets_status ON ip_assets(status, verification_status)",
		"CREATE INDEX IF NOT EXISTS idx_ip_assets_created_at ON ip_assets(created_at DESC)",

		// Product indexes
		"CREATE INDEX IF NOT EXISTS idx_products_creator ON products(creator_id)",
		"CREATE INDEX IF NOT EXISTS idx_products_category_status ON products(category, status)",
		"CREATE INDEX IF NOT EXISTS idx_products_price ON products(price)",
		"CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC)",

		// Transaction indexes
		"CREATE INDEX IF NOT EXISTS idx_transactions_buyer ON transactions(buyer_id)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_seller ON transactions(seller_id)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_type_status ON transactions(transaction_type, status)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC)",

		// License indexes
		"CREATE INDEX IF NOT EXISTS idx_license_applications_applicant ON license_applications(applicant_id)",
		"CREATE INDEX IF NOT EXISTS idx_license_applications_ip_asset ON license_applications(ip_asset_id)",
		"CREATE INDEX IF NOT EXISTS idx_license_applications_status ON license_applications(status)",

		// Admin indexes
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user_action ON audit_logs(user_id, action)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_admin_notifications_status ON admin_notifications(status, priority)",
		"CREATE INDEX IF NOT EXISTS idx_admin_notifications_type ON admin_notifications(type, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_content_reports_status ON content_reports(status, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_content_reports_type ON content_reports(reported_content_type, reported_content_id)",
		"CREATE INDEX IF NOT EXISTS idx_admin_settings_category ON admin_settings(category, key)",
		"CREATE INDEX IF NOT EXISTS idx_platform_analytics_metric ON platform_analytics(metric_name, metric_date)",
		"CREATE INDEX IF NOT EXISTS idx_platform_analytics_period ON platform_analytics(metric_period, metric_date DESC)",

		// Full-text search indexes
		"CREATE INDEX IF NOT EXISTS idx_ip_assets_search ON ip_assets USING GIN(to_tsvector('english', title || ' ' || description))",
		"CREATE INDEX IF NOT EXISTS idx_products_search ON products USING GIN(to_tsvector('english', title || ' ' || description))",
	}

	for _, index := range indexes {
		if err := db.Exec(index).Error; err != nil {
			log.Printf("Warning: Failed to create index: %s, Error: %v", index, err)
			// Continue with other indexes instead of failing completely
		}
	}

	return nil
}

// Seed initial data
func SeedInitialData(db *gorm.DB) error {
	log.Println("Seeding initial data...")

	// Create default admin user
	var adminCount int64
	db.Model(&models.User{}).Where("user_type = ?", models.UserTypeAdmin).Count(&adminCount)

	if adminCount == 0 {
		admin := &models.User{
			Username:          "admin",
			Email:             "admin@ipmarketplace.com",
			UserType:          models.UserTypeAdmin,
			VerificationLevel: models.VerificationLevelPremium,
			Status:            models.UserStatusActive,
			ProfileData: models.JSONB{
				"first_name": "System",
				"last_name":  "Administrator",
				"role":       "super_admin",
			},
		}

		if err := admin.SetPassword("admin123!@#"); err != nil {
			return fmt.Errorf("failed to set admin password: %w", err)
		}

		if err := db.Create(admin).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		log.Println("Default admin user created successfully")
	}

	// Create default platform settings
	defaultSettings := []models.AdminSettings{
		{
			Category:    "general",
			Key:         "platform_name",
			Value:       models.JSONB{"value": "IP Marketplace"},
			DataType:    "string",
			Description: "Platform name displayed to users",
		},
		{
			Category:    "general",
			Key:         "platform_description",
			Value:       models.JSONB{"value": "The world's first open IP authorization and secondary creation marketplace"},
			DataType:    "string",
			Description: "Platform description for marketing",
		},
		{
			Category:    "payments",
			Key:         "platform_fee_percentage",
			Value:       models.JSONB{"value": 5.0},
			DataType:    "float",
			Description: "Platform fee percentage for transactions",
		},
		{
			Category:    "payments",
			Key:         "minimum_payout",
			Value:       models.JSONB{"value": 10.0},
			DataType:    "float",
			Description: "Minimum amount for payout requests",
		},
		{
			Category:    "verification",
			Key:         "auto_verify_creators",
			Value:       models.JSONB{"value": false},
			DataType:    "boolean",
			Description: "Automatically verify new creators",
		},
		{
			Category:    "content",
			Key:         "auto_approve_ips",
			Value:       models.JSONB{"value": false},
			DataType:    "boolean",
			Description: "Automatically approve new IP assets",
		},
		{
			Category:    "content",
			Key:         "max_file_size",
			Value:       models.JSONB{"value": 50},
			DataType:    "integer",
			Description: "Maximum file size in MB for uploads",
		},
	}

	for _, setting := range defaultSettings {
		var count int64
		db.Model(&models.AdminSettings{}).Where("category = ? AND key = ?", setting.Category, setting.Key).Count(&count)

		if count == 0 {
			// Get admin user ID for the UpdatedBy field
			var admin models.User
			if err := db.Where("user_type = ?", models.UserTypeAdmin).First(&admin).Error; err == nil {
				setting.UpdatedBy = admin.ID
				if err := db.Create(&setting).Error; err != nil {
					log.Printf("Warning: Failed to create setting %s.%s: %v", setting.Category, setting.Key, err)
				}
			}
		}
	}

	log.Println("Initial data seeding completed")
	return nil
}

// Transaction helper
func WithTransaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
