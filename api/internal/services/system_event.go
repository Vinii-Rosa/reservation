package services

import (
	"encoding/json"
	"errors"
	"time"

	"reservation/api/internal/models"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("recurso não encontrado")

type ActorContext struct {
	Type     models.ActorType
	UserID   *string
	Name     string
	CompanyID string
}

type SystemEventService struct {
	db *gorm.DB
}

func NewSystemEventService(db *gorm.DB) *SystemEventService {
	return &SystemEventService{db: db}
}

func (s *SystemEventService) Log(actor ActorContext, eventType, resourceType, resourceID string, payload map[string]interface{}) error {
	var payloadJSON string
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadJSON = string(b)
	}

	event := models.SystemEvent{
		CompanyID:    companyIDPtr(actor.CompanyID),
		Type:         eventType,
		ActorType:    actor.Type,
		ActorUserID:  actor.UserID,
		ActorName:    actor.Name,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Payload:      payloadJSON,
		CreatedAt:    time.Now(),
	}
	return s.db.Create(&event).Error
}

func (s *SystemEventService) List(companyID, eventType, actorType string, from, to *time.Time) ([]models.SystemEvent, error) {
	q := s.db.Where("company_id = ?", companyID)
	if eventType != "" {
		q = q.Where("type = ?", eventType)
	}
	if actorType != "" {
		q = q.Where("actor_type = ?", actorType)
	}
	if from != nil {
		q = q.Where("created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("created_at <= ?", *to)
	}

	var events []models.SystemEvent
	err := q.Order("created_at DESC").Find(&events).Error
	return events, err
}
