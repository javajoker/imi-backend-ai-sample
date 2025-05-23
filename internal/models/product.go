// internal/models/product.go
package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Product struct {
	BaseModel
	CreatorID            uuid.UUID      `json:"creator_id" gorm:"type:uuid;not null;index"`
	LicenseID            uuid.UUID      `json:"license_id" gorm:"type:uuid;not null;index"`
	Title                string         `json:"title" gorm:"size:255;not null"`
	Description          string         `json:"description" gorm:"type:text"`
	Category             string         `json:"category" gorm:"size:100;index"`
	Price                float64        `json:"price" gorm:"type:decimal(10,2);not null"`
	InventoryCount       int            `json:"inventory_count" gorm:"default:0"`
	Images               pq.StringArray `json:"images" gorm:"type:text[]"`
	Specifications       JSONB          `json:"specifications" gorm:"type:jsonb"`
	Status               ProductStatus  `json:"status" gorm:"type:varchar(20);default:'draft';index"`
	AuthenticityVerified bool           `json:"authenticity_verified" gorm:"default:true"`
	Tags                 pq.StringArray `json:"tags" gorm:"type:text[]"`
	ViewCount            int64          `json:"view_count" gorm:"default:0"`
	SalesCount           int64          `json:"sales_count" gorm:"default:0"`
	Rating               float64        `json:"rating" gorm:"type:decimal(3,2);default:0"`
	ReviewCount          int64          `json:"review_count" gorm:"default:0"`

	// Relationships
	Creator      User                 `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	License      LicenseApplication   `json:"license,omitempty" gorm:"foreignKey:LicenseID"`
	Transactions []Transaction        `json:"transactions,omitempty" gorm:"foreignKey:ProductID"`
	AuthChain    []AuthorizationChain `json:"auth_chain,omitempty" gorm:"foreignKey:ProductID"`
}
