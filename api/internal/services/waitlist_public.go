package services

import (
	"math"

	"reservation/api/internal/models"
)

type JoinWaitlistInput struct {
	GuestName string           `json:"guest_name"`
	PartySize int              `json:"party_size"`
	NotifyVia models.NotifyVia `json:"notify_via"`
	Contact   string           `json:"contact"`
}

type WaitlistStatusResponse struct {
	Entry      models.WaitlistEntry `json:"entry"`
	Position   int                  `json:"position"`
	ETAMinutes int                  `json:"eta_minutes"`
}

func (s *WaitlistService) Join(token string, in JoinWaitlistInput) (*WaitlistStatusResponse, error) {
	company, err := getCompanyByWaitlistToken(s.db, token)
	if err != nil {
		return nil, err
	}

	entry := models.WaitlistEntry{
		CompanyID: company.ID,
		GuestName: in.GuestName,
		PartySize: in.PartySize,
		NotifyVia: in.NotifyVia,
		Contact:   in.Contact,
		Status:    models.WaitlistStatusWaiting,
	}
	if err := s.db.Create(&entry).Error; err != nil {
		return nil, err
	}

	_ = s.events.Log(ActorContext{
		Type: models.ActorTypeClient, Name: in.GuestName, CompanyID: company.ID,
	}, "waitlist_joined", "waitlist_entry", entry.ID, map[string]interface{}{
		"party_size": in.PartySize,
	})

	return s.buildStatus(&entry, company)
}

func (s *WaitlistService) GetStatus(token, entryID string) (*WaitlistStatusResponse, error) {
	company, err := getCompanyByWaitlistToken(s.db, token)
	if err != nil {
		return nil, err
	}
	var entry models.WaitlistEntry
	if err := s.db.Where("company_id = ? AND id = ?", company.ID, entryID).First(&entry).Error; err != nil {
		return nil, ErrNotFound
	}
	return s.buildStatus(&entry, company)
}

func (s *WaitlistService) buildStatus(entry *models.WaitlistEntry, company *models.Company) (*WaitlistStatusResponse, error) {
	position := 0
	if entry.Status == models.WaitlistStatusWaiting {
		var count int64
		s.db.Model(&models.WaitlistEntry{}).Where(
			"company_id = ? AND status = ? AND created_at < ?",
			company.ID, models.WaitlistStatusWaiting, entry.CreatedAt,
		).Count(&count)
		position = int(count) + 1
	}

	var tableCount int64
	s.db.Model(&models.Table{}).Where("company_id = ?", company.ID).Count(&tableCount)
	if tableCount == 0 {
		tableCount = 1
	}

	eta := int(math.Ceil(float64(position)/float64(tableCount))) * company.AvgTurnoverMinutes

	return &WaitlistStatusResponse{
		Entry: *entry, Position: position, ETAMinutes: eta,
	}, nil
}
