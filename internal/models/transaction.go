// internal/models/transaction.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	BaseModel
	TransactionType  TransactionType   `json:"transaction_type" gorm:"type:varchar(20);not null;index"`
	BuyerID          uuid.UUID         `json:"buyer_id" gorm:"type:uuid;not null;index"`
	SellerID         uuid.UUID         `json:"seller_id" gorm:"type:uuid;not null;index"`
	ProductID        *uuid.UUID        `json:"product_id" gorm:"type:uuid;index"`
	Amount           float64           `json:"amount" gorm:"type:decimal(10,2);not null"`
	PlatformFee      float64           `json:"platform_fee" gorm:"type:decimal(10,2);not null"`
	RevenueShares    JSONB             `json:"revenue_shares" gorm:"type:jsonb"`
	PaymentMethod    string            `json:"payment_method" gorm:"size:50"`
	PaymentReference string            `json:"payment_reference" gorm:"size:255"`
	Status           TransactionStatus `json:"status" gorm:"type:varchar(20);default:'pending';index"`
	ProcessedAt      *time.Time        `json:"processed_at"`
	RefundedAt       *time.Time        `json:"refunded_at"`
	RefundReason     string            `json:"refund_reason,omitempty" gorm:"type:text"`

	// Relationships
	Buyer   User     `json:"buyer,omitempty" gorm:"foreignKey:BuyerID"`
	Seller  User     `json:"seller,omitempty" gorm:"foreignKey:SellerID"`
	Product *Product `json:"product,omitempty" gorm:"foreignKey:ProductID"`
}

type AuthorizationChain struct {
	BaseModel
	ProductID        uuid.UUID  `json:"product_id" gorm:"type:uuid;not null;index"`
	IPAssetID        uuid.UUID  `json:"ip_asset_id" gorm:"type:uuid;not null;index"`
	LicenseID        uuid.UUID  `json:"license_id" gorm:"type:uuid;not null;index"`
	ParentChainID    *uuid.UUID `json:"parent_chain_id" gorm:"type:uuid;index"`
	BlockchainHash   string     `json:"blockchain_hash" gorm:"size:66"`
	VerificationCode string     `json:"verification_code" gorm:"size:32;uniqueIndex"`
	IsActive         bool       `json:"is_active" gorm:"default:true"`

	// Relationships
	Product     Product              `json:"product,omitempty" gorm:"foreignKey:ProductID"`
	IPAsset     IPAsset              `json:"ip_asset,omitempty" gorm:"foreignKey:IPAssetID"`
	License     LicenseApplication   `json:"license,omitempty" gorm:"foreignKey:LicenseID"`
	ParentChain *AuthorizationChain  `json:"parent_chain,omitempty" gorm:"foreignKey:ParentChainID"`
	ChildChains []AuthorizationChain `json:"child_chains,omitempty" gorm:"foreignKey:ParentChainID"`
}
