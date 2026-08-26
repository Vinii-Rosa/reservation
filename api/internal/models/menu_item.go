package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MenuItem struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID   string    `gorm:"type:uuid;not null;index" json:"company_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	PriceCents  int64     `gorm:"not null" json:"price_cents"`
	Active      bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (m *MenuItem) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
