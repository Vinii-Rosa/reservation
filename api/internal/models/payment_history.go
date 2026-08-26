package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentHistory struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID      string    `gorm:"type:uuid;not null;index" json:"company_id"`
	TableID        string    `gorm:"type:uuid;not null;index" json:"table_id"`
	TableNumber    string    `gorm:"not null" json:"table_number"`
	TableSessionID string    `gorm:"type:uuid;not null" json:"table_session_id"`
	TotalCents     int64     `gorm:"not null" json:"total_cents"`
	ItemsSnapshot  string    `gorm:"type:jsonb;not null" json:"items_snapshot"`
	PaidAt         time.Time `gorm:"not null" json:"paid_at"`
	CreatedAt      time.Time `json:"created_at"`
}

func (p *PaymentHistory) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
