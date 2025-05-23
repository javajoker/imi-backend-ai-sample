// internal/models/user.go
package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	BaseModel
	Username          string            `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email             string            `json:"email" gorm:"uniqueIndex;size:255;not null"`
	PasswordHash      string            `json:"-" gorm:"size:255;not null"`
	UserType          UserType          `json:"user_type" gorm:"type:varchar(20);not null"`
	VerificationLevel VerificationLevel `json:"verification_level" gorm:"type:varchar(20);default:'unverified'"`
	Status            UserStatus        `json:"status" gorm:"type:varchar(20);default:'active'"`
	ProfileData       JSONB             `json:"profile_data" gorm:"type:jsonb"`
	EmailVerifiedAt   *time.Time        `json:"email_verified_at"`
	LastLoginAt       *time.Time        `json:"last_login_at"`

	// Relationships
	IPAssets     []IPAsset            `json:"ip_assets,omitempty" gorm:"foreignKey:CreatorID"`
	Applications []LicenseApplication `json:"applications,omitempty" gorm:"foreignKey:ApplicantID"`
	Products     []Product            `json:"products,omitempty" gorm:"foreignKey:CreatorID"`
	Transactions []Transaction        `json:"transactions,omitempty" gorm:"foreignKey:BuyerID"`
}

func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}
