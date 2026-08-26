package services

import (
	"errors"
	"time"

	"reservation/api/internal/models"
)

type AvailabilitySlot struct {
	Time      time.Time `json:"time"`
	Available bool      `json:"available"`
}

func (s *ReservationService) CreatePublic(companyID string, in CreateReservationInput) (*models.Reservation, error) {
	if in.PartySize < 1 || in.GuestName == "" {
		return nil, errors.New("dados inválidos")
	}

	company, err := getCompany(s.db, companyID)
	if err != nil {
		return nil, err
	}

	if err := validateSchedule(company, in.ScheduledAt); err != nil {
		return nil, err
	}

	if err := s.checkCapacity(company, in.ScheduledAt, in.PartySize, ""); err != nil {
		return nil, err
	}

	res := models.Reservation{
		CompanyID:    companyID,
		GuestName:    in.GuestName,
		GuestContact: in.GuestContact,
		PartySize:    in.PartySize,
		ScheduledAt:  in.ScheduledAt,
		Status:       models.ReservationStatusPending,
	}
	if err := s.db.Create(&res).Error; err != nil {
		return nil, err
	}

	_ = s.events.Log(ActorContext{
		Type: models.ActorTypeClient, Name: in.GuestName, CompanyID: companyID,
	}, "table_reservation", "reservation", res.ID, map[string]interface{}{
		"party_size":   in.PartySize,
		"scheduled_at": in.ScheduledAt,
	})

	return &res, nil
}

func (s *ReservationService) GetByPublicToken(token string) (*models.Reservation, error) {
	var res models.Reservation
	if err := s.db.Where("public_token = ?", token).First(&res).Error; err != nil {
		return nil, ErrNotFound
	}
	return &res, nil
}

func (s *ReservationService) Availability(companyID string, date time.Time, partySize int) ([]AvailabilitySlot, error) {
	company, err := getCompany(s.db, companyID)
	if err != nil {
		return nil, err
	}

	slots := buildSlots(company, date)
	result := make([]AvailabilitySlot, 0, len(slots))
	for _, slot := range slots {
		err := s.checkCapacity(company, slot, partySize, "")
		result = append(result, AvailabilitySlot{Time: slot, Available: err == nil})
	}
	return result, nil
}

func (s *ReservationService) PublicSchedule(companyID string) (map[string]interface{}, error) {
	company, err := getCompany(s.db, companyID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"reservation_mode":      company.ReservationMode,
		"opens_at":              company.OpensAt,
		"closes_at":             company.ClosesAt,
		"fixed_time":            company.FixedTime,
		"slot_interval_minutes": company.SlotIntervalMinutes,
	}, nil
}
