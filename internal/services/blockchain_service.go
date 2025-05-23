// internal/services/blockchain_service.go
package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/config"
	"github.com/javajoker/imi-backend/internal/models"
)

type BlockchainService struct {
	db     *gorm.DB
	config *config.Config
}

type BlockchainRecord struct {
	Hash         string                 `json:"hash"`
	Timestamp    time.Time              `json:"timestamp"`
	Data         map[string]interface{} `json:"data"`
	PreviousHash string                 `json:"previous_hash"`
}

func NewBlockchainService(db *gorm.DB, config *config.Config) *BlockchainService {
	return &BlockchainService{
		db:     db,
		config: config,
	}
}

func (s *BlockchainService) CreateIPRecord(ipAssetID, creatorID uuid.UUID) (string, error) {
	// Get IP asset data
	var ipAsset models.IPAsset
	if err := s.db.First(&ipAsset, ipAssetID).Error; err != nil {
		return "", fmt.Errorf("IP asset not found: %w", err)
	}

	// Create blockchain record data
	recordData := map[string]interface{}{
		"type":       "ip_creation",
		"ip_id":      ipAssetID.String(),
		"creator_id": creatorID.String(),
		"title":      ipAsset.Title,
		"category":   ipAsset.Category,
		"timestamp":  time.Now().Unix(),
	}

	// Generate hash
	hash := s.generateHash(recordData)

	// TODO: In a real implementation, this would interact with actual blockchain
	// For now, we'll simulate by storing in database
	fmt.Printf("Blockchain record created for IP %s: %s\n", ipAssetID, hash)

	return hash, nil
}

func (s *BlockchainService) CreateLicenseRecord(licenseID, ipAssetID, applicantID uuid.UUID) (string, error) {
	recordData := map[string]interface{}{
		"type":         "license_grant",
		"license_id":   licenseID.String(),
		"ip_id":        ipAssetID.String(),
		"applicant_id": applicantID.String(),
		"timestamp":    time.Now().Unix(),
	}

	hash := s.generateHash(recordData)
	fmt.Printf("Blockchain record created for license %s: %s\n", licenseID, hash)

	return hash, nil
}

func (s *BlockchainService) CreateProductRecord(productID, licenseID uuid.UUID) (string, error) {
	recordData := map[string]interface{}{
		"type":       "product_creation",
		"product_id": productID.String(),
		"license_id": licenseID.String(),
		"timestamp":  time.Now().Unix(),
	}

	hash := s.generateHash(recordData)
	fmt.Printf("Blockchain record created for product %s: %s\n", productID, hash)

	return hash, nil
}

func (s *BlockchainService) VerifyChain(authChainID uuid.UUID) (bool, error) {
	// Get authorization chain
	var authChain models.AuthorizationChain
	if err := s.db.Preload("IPAsset").Preload("License").Preload("Product").
		First(&authChain, authChainID).Error; err != nil {
		return false, fmt.Errorf("authorization chain not found: %w", err)
	}

	// TODO: In a real implementation, this would verify the blockchain records
	// For now, we'll do basic validation
	if authChain.BlockchainHash == "" {
		return false, fmt.Errorf("no blockchain hash found")
	}

	if !authChain.IsActive {
		return false, fmt.Errorf("authorization chain is not active")
	}

	// Verify IP asset is approved
	if authChain.IPAsset.VerificationStatus != models.VerificationStatusApproved {
		return false, fmt.Errorf("IP asset is not approved")
	}

	// Verify license is active
	if authChain.License.Status != models.ApplicationStatusApproved {
		return false, fmt.Errorf("license is not approved")
	}

	return true, nil
}

func (s *BlockchainService) generateHash(data map[string]interface{}) string {
	// Convert data to JSON string for consistent hashing
	jsonStr := fmt.Sprintf("%+v", data)
	hash := sha256.Sum256([]byte(jsonStr))
	return hex.EncodeToString(hash[:])
}
