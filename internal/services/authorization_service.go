// internal/services/authorization_service.go
package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/utils"
)

type AuthorizationService struct {
	db                *gorm.DB
	blockchainService *BlockchainService
}

func NewAuthorizationService(db *gorm.DB, blockchainService *BlockchainService) *AuthorizationService {
	return &AuthorizationService{
		db:                db,
		blockchainService: blockchainService,
	}
}

func (s *AuthorizationService) CreateProductAuthChain(product *models.Product) (*models.AuthorizationChain, error) {
	// Generate unique verification code
	verificationCode, err := utils.GenerateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
	}

	// Create authorization chain record
	authChain := &models.AuthorizationChain{
		ProductID:        product.ID,
		IPAssetID:        product.License.IPAssetID,
		LicenseID:        product.LicenseID,
		VerificationCode: verificationCode,
		IsActive:         true,
	}

	// Create blockchain record
	if s.blockchainService != nil {
		if hash, err := s.blockchainService.CreateProductRecord(product.ID, product.LicenseID); err == nil {
			authChain.BlockchainHash = hash
		}
	}

	// Save to database
	if err := s.db.Create(authChain).Error; err != nil {
		return nil, fmt.Errorf("failed to create authorization chain: %w", err)
	}

	return authChain, nil
}

func (s *AuthorizationService) VerifyChain(authChainID uuid.UUID) (bool, error) {
	if s.blockchainService != nil {
		return s.blockchainService.VerifyChain(authChainID)
	}

	// Fallback verification without blockchain
	var authChain models.AuthorizationChain
	if err := s.db.Preload("IPAsset").Preload("License").Preload("Product").
		First(&authChain, authChainID).Error; err != nil {
		return false, fmt.Errorf("authorization chain not found: %w", err)
	}

	return authChain.IsActive, nil
}

func (s *AuthorizationService) VerifyProductByCode(verificationCode string) (*models.AuthorizationChain, error) {
	var authChain models.AuthorizationChain
	if err := s.db.Where("verification_code = ? AND is_active = ?", verificationCode, true).
		Preload("Product").Preload("IPAsset").Preload("License").
		First(&authChain).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid verification code")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify the chain is still valid
	if valid, err := s.VerifyChain(authChain.ID); err != nil || !valid {
		return nil, errors.New("authorization chain verification failed")
	}

	return &authChain, nil
}

func (s *AuthorizationService) RevokeAuthChain(authChainID uuid.UUID, reason string) error {
	var authChain models.AuthorizationChain
	if err := s.db.First(&authChain, authChainID).Error; err != nil {
		return fmt.Errorf("authorization chain not found: %w", err)
	}

	// Deactivate the chain
	authChain.IsActive = false

	if err := s.db.Save(&authChain).Error; err != nil {
		return fmt.Errorf("failed to revoke authorization chain: %w", err)
	}

	return nil
}

func (s *AuthorizationService) GetAuthChainHistory(productID uuid.UUID) ([]models.AuthorizationChain, error) {
	var chains []models.AuthorizationChain
	if err := s.db.Where("product_id = ?", productID).
		Preload("IPAsset").Preload("License").
		Order("created_at DESC").
		Find(&chains).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch authorization chain history: %w", err)
	}

	return chains, nil
}
