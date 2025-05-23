// internal/models/license.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type LicenseApplication struct {
	BaseModel
	IPAssetID       uuid.UUID         `json:"ip_asset_id" gorm:"type:uuid;not null;index"`
	ApplicantID     uuid.UUID         `json:"applicant_id" gorm:"type:uuid;not null;index"`
	LicenseTermsID  uuid.UUID         `json:"license_terms_id" gorm:"type:uuid;not null;index"`
	ApplicationData JSONB             `json:"application_data" gorm:"type:jsonb"`
	Status          ApplicationStatus `json:"status" gorm:"type:varchar(20);default:'pending';index"`
	ApprovedAt      *time.Time        `json:"approved_at"`
	ApprovedBy      *uuid.UUID        `json:"approved_by" gorm:"type:uuid"`
	RejectionReason string            `json:"rejection_reason,omitempty" gorm:"type:text"`
	ExpiresAt       *time.Time        `json:"expires_at"`
	IsActive        bool              `json:"is_active" gorm:"default:true"`

	// Relationships
	IPAsset      IPAsset      `json:"ip_asset,omitempty" gorm:"foreignKey:IPAssetID"`
	Applicant    User         `json:"applicant,omitempty" gorm:"foreignKey:ApplicantID"`
	LicenseTerms LicenseTerms `json:"license_terms,omitempty" gorm:"foreignKey:LicenseTermsID"`
	Approver     *User        `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`
	Products     []Product    `json:"products,omitempty" gorm:"foreignKey:LicenseID"`
}
