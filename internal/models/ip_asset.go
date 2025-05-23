// internal/models/ip_asset.go
package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type IPAsset struct {
	BaseModel
	CreatorID          uuid.UUID          `json:"creator_id" gorm:"type:uuid;not null;index"`
	Title              string             `json:"title" gorm:"size:255;not null"`
	Description        string             `json:"description" gorm:"type:text"`
	Category           string             `json:"category" gorm:"size:100;index"`
	ContentType        string             `json:"content_type" gorm:"size:50"`
	FileURLs           pq.StringArray     `json:"file_urls" gorm:"type:text[]"`
	Metadata           JSONB              `json:"metadata" gorm:"type:jsonb"`
	VerificationStatus VerificationStatus `json:"verification_status" gorm:"type:varchar(20);default:'pending';index"`
	BlockchainHash     string             `json:"blockchain_hash,omitempty" gorm:"size:66"`
	Status             ProductStatus      `json:"status" gorm:"type:varchar(20);default:'active';index"`
	Tags               pq.StringArray     `json:"tags" gorm:"type:text[]"`
	ViewCount          int64              `json:"view_count" gorm:"default:0"`
	LikeCount          int64              `json:"like_count" gorm:"default:0"`

	// Relationships
	Creator      User                 `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	LicenseTerms []LicenseTerms       `json:"license_terms,omitempty" gorm:"foreignKey:IPAssetID"`
	Applications []LicenseApplication `json:"applications,omitempty" gorm:"foreignKey:IPAssetID"`
}

type LicenseTerms struct {
	BaseModel
	IPAssetID              uuid.UUID   `json:"ip_asset_id" gorm:"type:uuid;not null;index"`
	LicenseType            LicenseType `json:"license_type" gorm:"type:varchar(20);not null"`
	RevenueSharePercentage float64     `json:"revenue_share_percentage" gorm:"type:decimal(5,2);not null"`
	BaseFee                float64     `json:"base_fee" gorm:"type:decimal(10,2);default:0"`
	Territory              string      `json:"territory" gorm:"size:100;default:'global'"`
	Duration               string      `json:"duration" gorm:"size:50;default:'perpetual'"`
	Requirements           string      `json:"requirements" gorm:"type:text"`
	Restrictions           string      `json:"restrictions" gorm:"type:text"`
	AutoApprove            bool        `json:"auto_approve" gorm:"default:false"`
	MaxLicenses            int         `json:"max_licenses" gorm:"default:0"` // 0 = unlimited
	IsActive               bool        `json:"is_active" gorm:"default:true"`

	// Relationships
	IPAsset      IPAsset              `json:"ip_asset,omitempty" gorm:"foreignKey:IPAssetID"`
	Applications []LicenseApplication `json:"applications,omitempty" gorm:"foreignKey:LicenseTermsID"`
}
