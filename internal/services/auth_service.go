// internal/services/auth_service.go
package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/config"
	"github.com/javajoker/imi-backend/internal/models"
	"github.com/javajoker/imi-backend/internal/utils"
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RegisterRequest struct {
	Username    string                 `json:"username" validate:"required,username"`
	Email       string                 `json:"email" validate:"required,email"`
	Password    string                 `json:"password" validate:"required,strong_password"`
	UserType    models.UserType        `json:"user_type" validate:"required"`
	ProfileData map[string]interface{} `json:"profile_data,omitempty"`
}

type AuthResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"` // in seconds
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,strong_password"`
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		db:  db,
		cfg: cfg,
	}
}

func (s *AuthService) Register(req *RegisterRequest) (*AuthResponse, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		if existingUser.Email == req.Email {
			return nil, errors.New("user with this email already exists")
		}
		return nil, errors.New("username already taken")
	}

	// Validate user type
	if req.UserType != models.UserTypeCreator &&
		req.UserType != models.UserTypeSecondaryCreator &&
		req.UserType != models.UserTypeBuyer {
		return nil, errors.New("invalid user type")
	}

	// Create new user
	user := &models.User{
		Username:    req.Username,
		Email:       req.Email,
		UserType:    req.UserType,
		Status:      models.UserStatusActive,
		ProfileData: models.JSONB(req.ProfileData),
	}

	// Set password
	if err := user.SetPassword(req.Password); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Save user
	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := utils.GenerateJWT(
		user.ID,
		user.Username,
		string(user.UserType),
		string(user.VerificationLevel),
		s.cfg.JWT.AccessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, s.cfg.JWT.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Send verification email (async)
	go s.sendVerificationEmail(user)

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.cfg.JWT.AccessTokenTTL * 3600, // Convert hours to seconds
	}, nil
}

func (s *AuthService) Login(req *LoginRequest) (*AuthResponse, error) {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Find user by email
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check if user is suspended or banned
	if user.Status == models.UserStatusSuspended {
		return nil, errors.New("account is suspended")
	}
	if user.Status == models.UserStatusBanned {
		return nil, errors.New("account is banned")
	}

	// Verify password
	if err := user.CheckPassword(req.Password); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Save(&user)

	// Generate tokens
	accessToken, err := utils.GenerateJWT(
		user.ID,
		user.Username,
		string(user.UserType),
		string(user.VerificationLevel),
		s.cfg.JWT.AccessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, s.cfg.JWT.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &AuthResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.cfg.JWT.AccessTokenTTL * 3600,
	}, nil
}

func (s *AuthService) RefreshToken(refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	userIDStr, err := utils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	// Find user
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check user status
	if user.Status != models.UserStatusActive {
		return nil, errors.New("account is not active")
	}

	// Generate new tokens
	accessToken, err := utils.GenerateJWT(
		user.ID,
		user.Username,
		string(user.UserType),
		string(user.VerificationLevel),
		s.cfg.JWT.AccessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := utils.GenerateRefreshToken(user.ID, s.cfg.JWT.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &AuthResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.cfg.JWT.AccessTokenTTL * 3600,
	}, nil
}

func (s *AuthService) ForgotPassword(req *ForgotPasswordRequest) error {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Find user by email
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Don't reveal if email exists or not for security
		return nil
	}

	// Generate reset token
	resetToken, err := utils.GenerateVerificationCode()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// TODO: Store reset token (you might want to create a separate table for this)
	// For now, we'll use the profile_data field
	if user.ProfileData == nil {
		user.ProfileData = make(models.JSONB)
	}
	user.ProfileData["reset_token"] = resetToken
	user.ProfileData["reset_token_expires"] = time.Now().Add(1 * time.Hour).Unix()

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	// Send reset email (async)
	go s.sendPasswordResetEmail(&user, resetToken)

	return nil
}

func (s *AuthService) ResetPassword(req *ResetPasswordRequest) error {
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Find user with reset token
	var user models.User
	if err := s.db.Where("profile_data->>'reset_token' = ?", req.Token).First(&user).Error; err != nil {
		return errors.New("invalid or expired reset token")
	}

	// Check token expiration
	if expiresAt, ok := user.ProfileData["reset_token_expires"].(float64); ok {
		if time.Now().Unix() > int64(expiresAt) {
			return errors.New("reset token has expired")
		}
	} else {
		return errors.New("invalid reset token")
	}

	// Update password
	if err := user.SetPassword(req.NewPassword); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Clear reset token
	delete(user.ProfileData, "reset_token")
	delete(user.ProfileData, "reset_token_expires")

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func (s *AuthService) VerifyEmail(token string) error {
	// Find user with verification token
	var user models.User
	if err := s.db.Where("profile_data->>'email_verification_token' = ?", token).First(&user).Error; err != nil {
		return errors.New("invalid verification token")
	}

	// Check if already verified
	if user.EmailVerifiedAt != nil {
		return errors.New("email already verified")
	}

	// Mark email as verified
	now := time.Now()
	user.EmailVerifiedAt = &now

	// Clear verification token
	if user.ProfileData == nil {
		user.ProfileData = make(models.JSONB)
	}
	delete(user.ProfileData, "email_verification_token")

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return nil
}

func (s *AuthService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

func (s *AuthService) sendVerificationEmail(user *models.User) {
	// Generate verification token
	token, err := utils.GenerateVerificationCode()
	if err != nil {
		return
	}

	// Store verification token
	if user.ProfileData == nil {
		user.ProfileData = make(models.JSONB)
	}
	user.ProfileData["email_verification_token"] = token
	s.db.Save(user)

	// TODO: Send actual email using your email service
	// For now, just log the verification URL
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.cfg.Frontend.BaseURL, token)
	fmt.Printf("Email verification URL for %s: %s\n", user.Email, verificationURL)
}

func (s *AuthService) sendPasswordResetEmail(user *models.User, token string) {
	// TODO: Send actual email using your email service
	// For now, just log the reset URL
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.Frontend.BaseURL, token)
	fmt.Printf("Password reset URL for %s: %s\n", user.Email, resetURL)
}
