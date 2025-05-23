// internal/models/admin.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type AdminSettings struct {
	BaseModel
	Category    string    `json:"category" gorm:"size:50;not null;index"`
	Key         string    `json:"key" gorm:"size:100;not null;index"`
	Value       JSONB     `json:"value" gorm:"type:jsonb;not null"`
	DataType    string    `json:"data_type" gorm:"size:20;not null"`
	Description string    `json:"description" gorm:"type:text"`
	UpdatedBy   uuid.UUID `json:"updated_by" gorm:"type:uuid;not null"`

	// Relationships
	UpdatedByUser User `json:"updated_by_user,omitempty" gorm:"foreignKey:UpdatedBy"`
}

type AuditLog struct {
	BaseModel
	UserID       *uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	Action       string     `json:"action" gorm:"size:100;not null;index"`
	ResourceType string     `json:"resource_type" gorm:"size:50;not null;index"`
	ResourceID   *uuid.UUID `json:"resource_id" gorm:"type:uuid;index"`
	OldValues    JSONB      `json:"old_values" gorm:"type:jsonb"`
	NewValues    JSONB      `json:"new_values" gorm:"type:jsonb"`
	IPAddress    string     `json:"ip_address" gorm:"size:45"`
	UserAgent    string     `json:"user_agent" gorm:"type:text"`

	// Relationships
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type AdminNotification struct {
	BaseModel
	Type                string     `json:"type" gorm:"type:varchar(50);not null;index"`
	Title               string     `json:"title" gorm:"size:255;not null"`
	Message             string     `json:"message" gorm:"type:text;not null"`
	Priority            string     `json:"priority" gorm:"type:varchar(20);default:'medium';index"`
	Status              string     `json:"status" gorm:"type:varchar(20);default:'unread';index"`
	RelatedResourceType string     `json:"related_resource_type,omitempty" gorm:"size:50"`
	RelatedResourceID   *uuid.UUID `json:"related_resource_id" gorm:"type:uuid"`
	ReadAt              *time.Time `json:"read_at"`
}

type ContentReport struct {
	BaseModel
	ReporterID          uuid.UUID  `json:"reporter_id" gorm:"type:uuid;not null;index"`
	ReportedContentType string     `json:"reported_content_type" gorm:"type:varchar(20);not null;index"`
	ReportedContentID   uuid.UUID  `json:"reported_content_id" gorm:"type:uuid;not null;index"`
	Reason              string     `json:"reason" gorm:"size:100;not null"`
	Description         string     `json:"description" gorm:"type:text"`
	Status              string     `json:"status" gorm:"type:varchar(20);default:'pending';index"`
	AdminNotes          string     `json:"admin_notes,omitempty" gorm:"type:text"`
	ResolvedBy          *uuid.UUID `json:"resolved_by" gorm:"type:uuid"`
	ResolvedAt          *time.Time `json:"resolved_at"`

	// Relationships
	Reporter User  `json:"reporter,omitempty" gorm:"foreignKey:ReporterID"`
	Resolver *User `json:"resolver,omitempty" gorm:"foreignKey:ResolvedBy"`
}

type PlatformAnalytics struct {
	BaseModel
	MetricName     string    `json:"metric_name" gorm:"size:100;not null;index"`
	MetricValue    float64   `json:"metric_value" gorm:"type:decimal(15,2);not null"`
	MetricDate     time.Time `json:"metric_date" gorm:"type:date;not null;index"`
	MetricPeriod   string    `json:"metric_period" gorm:"type:varchar(20);not null;index"`
	AdditionalData JSONB     `json:"additional_data" gorm:"type:jsonb"`
}
