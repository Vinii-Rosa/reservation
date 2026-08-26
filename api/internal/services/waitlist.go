package services

import (
	"fmt"
	"time"

	"reservation/api/internal/config"
	"reservation/api/internal/models"
	"reservation/api/internal/notify"

	"gorm.io/gorm"
)

type WaitlistService struct {
	db       *gorm.DB
	cfg      config.Config
	notifier notify.Notifier
	events   *SystemEventService
	tables   *TableService
}

func NewWaitlistService(db *gorm.DB, cfg config.Config, notifier notify.Notifier, events *SystemEventService, tables *TableService) *WaitlistService {
	return &WaitlistService{db: db, cfg: cfg, notifier: notifier, events: events, tables: tables}
}

func (s *WaitlistService) List(companyID string) ([]models.WaitlistEntry, error) {
	var entries []models.WaitlistEntry
	err := s.db.Where("company_id = ? AND status IN ?", companyID, []models.WaitlistStatus{
		models.WaitlistStatusWaiting, models.WaitlistStatusCalled,
	}).Order("created_at ASC").Find(&entries).Error
	return entries, err
}

func (s *WaitlistService) UpdateStatus(companyID string, actor ActorContext, id string, status models.WaitlistStatus) (*models.WaitlistEntry, error) {
	var entry models.WaitlistEntry
	if err := s.db.Where("company_id = ? AND id = ?", companyID, id).First(&entry).Error; err != nil {
		return nil, ErrNotFound
	}
	entry.Status = status
	if status == models.WaitlistStatusCalled {
		now := time.Now()
		entry.CalledAt = &now
	}
	if err := s.db.Save(&entry).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "waitlist_updated", "waitlist_entry", id, map[string]interface{}{"status": status})
	return &entry, nil
}

func (s *WaitlistService) OnTableFreed(companyID string) error {
	company, err := getCompany(s.db, companyID)
	if err != nil {
		return err
	}

	if err := s.expireTimedOutCalls(company.ID); err != nil {
		return err
	}

	var tables []models.Table
	s.db.Where("company_id = ? AND status = ?", companyID, models.TableStatusAvailable).Find(&tables)
	if len(tables) == 0 {
		return nil
	}

	var entry models.WaitlistEntry
	if err := s.db.Where("company_id = ? AND status = ?", companyID, models.WaitlistStatusWaiting).
		Order("created_at ASC").First(&entry).Error; err != nil {
		return nil
	}

	for _, t := range tables {
		if t.Capacity >= entry.PartySize {
			now := time.Now()
			entry.Status = models.WaitlistStatusCalled
			entry.CalledAt = &now
			if err := s.db.Save(&entry).Error; err != nil {
				return err
			}
			s.notifyCalled(&entry, company)
			return nil
		}
	}
	return nil
}

func (s *WaitlistService) expireTimedOutCalls(companyID string) error {
	timeout := time.Duration(s.cfg.WaitlistCallTimeoutMinutes) * time.Minute
	cutoff := time.Now().Add(-timeout)

	var expired []models.WaitlistEntry
	if err := s.db.Where("company_id = ? AND status = ? AND called_at < ?", companyID, models.WaitlistStatusCalled, cutoff).
		Find(&expired).Error; err != nil {
		return err
	}

	for _, e := range expired {
		e.Status = models.WaitlistStatusExpired
		if err := s.db.Save(&e).Error; err != nil {
			return err
		}
	}
	if len(expired) > 0 {
		return s.OnTableFreed(companyID)
	}
	return nil
}

func (s *WaitlistService) notifyCalled(entry *models.WaitlistEntry, company *models.Company) {
	msg := fmt.Sprintf("Olá %s, sua mesa no %s está pronta!", entry.GuestName, company.Name)
	if entry.NotifyVia == models.NotifyViaEmail {
		_ = s.notifier.SendEmail(entry.Contact, "Sua vez chegou!", msg)
		return
	}
	_ = s.notifier.SendWhatsApp(entry.Contact, msg)
}
