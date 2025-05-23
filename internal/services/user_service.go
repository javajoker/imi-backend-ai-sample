// internal/services/user_service.go
package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/utils"
)

type UserService struct {
	db             *gorm.DB
	storageService *StorageService
}

type UpdateUserProfileRequest struct {
	Username    string                 `json:"username,omitempty" validate:"omitempty,username"`
	ProfileData map[string]interface{} `json:"profile_data,omitempty"`
}

func NewUserService(db *gorm.DB, storageService *StorageService) *UserService {
	return &UserService{
		db:             db,
		storageService: storageService,
	}
}

func (s *UserService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

func (s *UserService) GetPublicProfile(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.Select("id, username, user_type, verification_level, profile_data, created_at").
		First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

func (s *UserService) UpdateProfile(userID uuid.UUID, req *UpdateUserProfileRequest) (*models.User, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Find user
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check username uniqueness if updating
	if req.Username != "" && req.Username != user.Username {
		var existingUser models.User
		if err := s.db.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser).Error; err == nil {
			return nil, errors.New("username already taken")
		}
	}

	// Update fields
	if req.Username != "" {
		user.Username = req.Username
	}

	if req.ProfileData != nil {
		if user.ProfileData == nil {
			user.ProfileData = make(models.JSONB)
		}
		// Merge with existing profile data
		for key, value := range req.ProfileData {
			user.ProfileData[key] = value
		}
	}

	// Save changes
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return &user, nil
}

func (s *UserService) DeleteAccount(userID uuid.UUID, password string) error {
	// Find user
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify password
	if err := user.CheckPassword(password); err != nil {
		return errors.New("invalid password")
	}

	// Check for active licenses or products
	var licenseCount, productCount int64
	s.db.Model(&models.LicenseApplication{}).
		Where("applicant_id = ? AND status = ?", userID, models.ApplicationStatusApproved).
		Count(&licenseCount)

	s.db.Model(&models.Product{}).
		Where("creator_id = ? AND status = ?", userID, models.ProductStatusActive).
		Count(&productCount)

	if licenseCount > 0 || productCount > 0 {
		return errors.New("cannot delete account with active licenses or products")
	}

	// Soft delete user
	if err := s.db.Delete(&user).Error; err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}
