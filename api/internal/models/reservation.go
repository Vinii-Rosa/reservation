package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusSeated    ReservationStatus = "seated"
	ReservationStatusCompleted ReservationStatus = "completed"
	ReservationStatusCancelled ReservationStatus = "cancelled"
	ReservationStatusNoShow    ReservationStatus = "no_show"
)

type Reservation struct {
	ID           string            `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID    string            `gorm:"type:uuid;not null;index" json:"company_id"`
	TableID      *string           `gorm:"type:uuid;index" json:"table_id,omitempty"`
	GuestName    string            `gorm:"not null" json:"guest_name"`
	GuestContact string            `gorm:"not null" json:"guest_contact"`
	PartySize    int               `gorm:"not null" json:"party_size"`
	ScheduledAt  time.Time         `gorm:"not null;index" json:"scheduled_at"`
	Status       ReservationStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	PublicToken  string            `gorm:"type:varchar(64);uniqueIndex;not null" json:"public_token"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func (r *Reservation) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.PublicToken == "" {
		r.PublicToken = uuid.NewString()
	}
	return nil
}
