package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TableStatus string

const (
	TableStatusAvailable TableStatus = "available"
	TableStatusOccupied  TableStatus = "occupied"
)

type Table struct {
	ID          string      `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID   string      `gorm:"type:uuid;not null;index" json:"company_id"`
	TableNumber string      `gorm:"not null" json:"table_number"`
	Capacity    int         `gorm:"not null" json:"capacity"`
	Status      TableStatus `gorm:"type:varchar(20);not null;default:'available'" json:"status"`
	PublicToken string      `gorm:"type:varchar(64);uniqueIndex;not null" json:"public_token"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func (t *Table) BeforeCreate(_ *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.PublicToken == "" {
		t.PublicToken = uuid.NewString()
	}
	return nil
}
