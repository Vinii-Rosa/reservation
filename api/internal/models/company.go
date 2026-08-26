package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReservationMode string

const (
	ReservationModeRange ReservationMode = "range"
	ReservationModeFixed ReservationMode = "fixed"
)

type DocumentType string

const (
	DocumentCNPJ DocumentType = "cnpj"
	DocumentCPF  DocumentType = "cpf"
)

type Address struct {
	ZipCode      string `gorm:"type:varchar(8)" json:"zip_code"`
	Street       string `json:"street"`
	Number       string `json:"number"`
	Complement   string `json:"complement"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	State        string `gorm:"type:varchar(2)" json:"state"`
}

type Company struct {
	ID                  string          `gorm:"type:uuid;primaryKey" json:"id"`
	Name                string          `gorm:"not null" json:"name"`
	DocumentType        DocumentType    `gorm:"type:varchar(4);not null" json:"document_type"`
	Document            string          `gorm:"type:varchar(14);not null;uniqueIndex" json:"document"`
	Email               string          `gorm:"not null;uniqueIndex" json:"email"`
	Phone               string          `gorm:"not null" json:"phone"`
	Address             Address         `gorm:"embedded;embeddedPrefix:address_" json:"address"`
	ProfilePhotoURL     string          `json:"profile_photo_url"`
	ReservationMode     ReservationMode `gorm:"type:varchar(20);not null;default:'range'" json:"reservation_mode"`
	OpensAt             string          `gorm:"type:varchar(5);default:'18:00'" json:"opens_at"`
	ClosesAt            string          `gorm:"type:varchar(5);default:'23:00'" json:"closes_at"`
	FixedTime           string          `gorm:"type:varchar(5)" json:"fixed_time"`
	SlotIntervalMinutes int             `gorm:"not null;default:30" json:"slot_interval_minutes"`
	AvgTurnoverMinutes  int             `gorm:"not null;default:90" json:"avg_turnover_minutes"`
	WaitlistToken       string          `gorm:"type:varchar(64);uniqueIndex;not null" json:"waitlist_token"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

func (c *Company) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.WaitlistToken == "" {
		c.WaitlistToken = uuid.NewString()
	}
	return nil
}
