package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeClient ActorType = "client"
	ActorTypeSystem ActorType = "system"
)

type SystemEvent struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID    *string   `gorm:"type:uuid;index" json:"company_id,omitempty"`
	Type         string    `gorm:"type:varchar(64);not null;index" json:"type"`
	ActorType    ActorType `gorm:"type:varchar(20);not null" json:"actor_type"`
	ActorUserID  *string   `gorm:"type:uuid;index" json:"actor_user_id,omitempty"`
	ActorName    string    `json:"actor_name,omitempty"`
	ResourceType string    `gorm:"type:varchar(64)" json:"resource_type,omitempty"`
	ResourceID   string    `gorm:"type:uuid" json:"resource_id,omitempty"`
	Payload      string    `gorm:"type:jsonb" json:"payload,omitempty"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

func (e *SystemEvent) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
