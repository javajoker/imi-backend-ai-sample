// internal/models/common.go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base model with common fields
type BaseModel struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// JSONB type for PostgreSQL
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// Enums
type UserType string

const (
	UserTypeCreator          UserType = "creator"
	UserTypeSecondaryCreator UserType = "secondary_creator"
	UserTypeBuyer            UserType = "buyer"
	UserTypeAdmin            UserType = "admin"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"
)

type VerificationLevel string

const (
	VerificationLevelUnverified VerificationLevel = "unverified"
	VerificationLevelVerified   VerificationLevel = "verified"
	VerificationLevelPremium    VerificationLevel = "premium"
)

type VerificationStatus string

const (
	VerificationStatusPending  VerificationStatus = "pending"
	VerificationStatusApproved VerificationStatus = "approved"
	VerificationStatusRejected VerificationStatus = "rejected"
)

type LicenseType string

const (
	LicenseTypeStandard  LicenseType = "standard"
	LicenseTypePremium   LicenseType = "premium"
	LicenseTypeExclusive LicenseType = "exclusive"
)

type ApplicationStatus string

const (
	ApplicationStatusPending  ApplicationStatus = "pending"
	ApplicationStatusApproved ApplicationStatus = "approved"
	ApplicationStatusRejected ApplicationStatus = "rejected"
	ApplicationStatusRevoked  ApplicationStatus = "revoked"
)

type ProductStatus string

const (
	ProductStatusDraft     ProductStatus = "draft"
	ProductStatusActive    ProductStatus = "active"
	ProductStatusSoldOut   ProductStatus = "sold_out"
	ProductStatusSuspended ProductStatus = "suspended"
)

type TransactionType string

const (
	TransactionTypeProductSale  TransactionType = "product_sale"
	TransactionTypeLicenseFee   TransactionType = "license_fee"
	TransactionTypeRevenueShare TransactionType = "revenue_share"
)

type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusRefunded  TransactionStatus = "refunded"
)
