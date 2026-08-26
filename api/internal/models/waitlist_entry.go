package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotifyVia string

const (
	NotifyViaEmail    NotifyVia = "email"
	NotifyViaWhatsApp NotifyVia = "whatsapp"
)

type WaitlistStatus string

const (
	WaitlistStatusWaiting   WaitlistStatus = "waiting"
	WaitlistStatusCalled    WaitlistStatus = "called"
	WaitlistStatusSeated    WaitlistStatus = "seated"
	WaitlistStatusCancelled WaitlistStatus = "cancelled"
	WaitlistStatusExpired   WaitlistStatus = "expired"
)

type WaitlistEntry struct {
	ID        string         `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID string         `gorm:"type:uuid;not null;index" json:"company_id"`
	GuestName string         `gorm:"not null" json:"guest_name"`
	PartySize int            `gorm:"not null" json:"party_size"`
	NotifyVia NotifyVia      `gorm:"type:varchar(20);not null" json:"notify_via"`
	Contact   string         `gorm:"not null" json:"contact"`
	Status    WaitlistStatus `gorm:"type:varchar(20);not null;default:'waiting'" json:"status"`
	CalledAt  *time.Time     `json:"called_at,omitempty"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (w *WaitlistEntry) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
