// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	Redis       RedisConfig
	AWS         AWSConfig
	Blockchain  BlockchainConfig
	Payment     PaymentConfig
	Email       EmailConfig
	I18n        I18nConfig
	Frontend    FrontendConfig
}

type FrontendConfig struct {
	BaseURL string
}

type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

type DatabaseConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Database     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  int
	LogLevel     string
}

type JWTConfig struct {
	SecretKey       string
	AccessTokenTTL  int // in hours
	RefreshTokenTTL int // in hours
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3Bucket        string
	CloudFrontURL   string
}

type BlockchainConfig struct {
	Network         string
	RPC_URL         string
	PrivateKey      string
	ContractAddress string
}

type PaymentConfig struct {
	StripeSecretKey      string
	StripePublishableKey string
	PayPalClientID       string
	PayPalClientSecret   string
	PlatformFeePercent   float64
	MinimumPayout        float64
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type I18nConfig struct {
	DefaultLocale string
	LocalesPath   string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	config := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "localhost"),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 15),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 15),
			IdleTimeout:  getEnvAsInt("SERVER_IDLE_TIMEOUT", 60),
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnv("DB_PORT", "5432"),
			User:         getEnv("DB_USER", "postgres"),
			Password:     getEnv("DB_PASSWORD", ""),
			Database:     getEnv("DB_NAME", "ip_marketplace"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
			MaxLifetime:  getEnvAsInt("DB_MAX_LIFETIME", 300),
		},
		JWT: JWTConfig{
			SecretKey:       getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessTokenTTL:  getEnvAsInt("JWT_ACCESS_TTL", 24),   // 24 hours
			RefreshTokenTTL: getEnvAsInt("JWT_REFRESH_TTL", 168), // 7 days
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			S3Bucket:        getEnv("AWS_S3_BUCKET", "ip-marketplace-assets"),
			CloudFrontURL:   getEnv("AWS_CLOUDFRONT_URL", ""),
		},
		Blockchain: BlockchainConfig{
			Network:         getEnv("BLOCKCHAIN_NETWORK", "polygon"),
			RPC_URL:         getEnv("BLOCKCHAIN_RPC_URL", ""),
			PrivateKey:      getEnv("BLOCKCHAIN_PRIVATE_KEY", ""),
			ContractAddress: getEnv("BLOCKCHAIN_CONTRACT_ADDRESS", ""),
		},
		Payment: PaymentConfig{
			StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
			PayPalClientID:       getEnv("PAYPAL_CLIENT_ID", ""),
			PayPalClientSecret:   getEnv("PAYPAL_CLIENT_SECRET", ""),
			PlatformFeePercent:   getEnvAsFloat("PLATFORM_FEE_PERCENT", 5.0),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getEnv("SMTP_PORT", "587"),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "noreply@ipmarketplace.com"),
			FromName:     getEnv("FROM_NAME", "IP Marketplace"),
		},
		I18n: I18nConfig{
			DefaultLocale: getEnv("DEFAULT_LOCALE", "en"),
			LocalesPath:   getEnv("LOCALES_PATH", "./internal/i18n/locales"),
		},
	}

	return config, config.Validate()
}

func (c *Config) Validate() error {
	if c.JWT.SecretKey == "your-secret-key-change-in-production" && c.Environment == "production" {
		return fmt.Errorf("JWT secret key must be changed in production")
	}

	if c.Database.Password == "" && c.Environment == "production" {
		return fmt.Errorf("database password is required in production")
	}

	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(strings.ToLower(value)); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
