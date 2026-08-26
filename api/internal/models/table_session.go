package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TableSession struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID string     `gorm:"type:uuid;not null;index" json:"company_id"`
	TableID   string     `gorm:"type:uuid;not null;index" json:"table_id"`
	OpenedAt  time.Time  `gorm:"not null" json:"opened_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (ts *TableSession) BeforeCreate(_ *gorm.DB) error {
	if ts.ID == "" {
		ts.ID = uuid.NewString()
	}
	return nil
}
