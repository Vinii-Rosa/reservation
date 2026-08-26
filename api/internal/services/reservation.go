package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"reservation/api/internal/models"

	"gorm.io/gorm"
)

type ReservationService struct {
	db     *gorm.DB
	events *SystemEventService
}

func NewReservationService(db *gorm.DB, events *SystemEventService) *ReservationService {
	return &ReservationService{db: db, events: events}
}

type CreateReservationInput struct {
	GuestName    string    `json:"guest_name"`
	GuestContact string    `json:"guest_contact"`
	PartySize    int       `json:"party_size"`
	ScheduledAt  time.Time `json:"scheduled_at"`
}

type UpdateReservationInput struct {
	GuestName    *string                   `json:"guest_name"`
	GuestContact *string                   `json:"guest_contact"`
	PartySize    *int                      `json:"party_size"`
	ScheduledAt  *time.Time                `json:"scheduled_at"`
	Status       *models.ReservationStatus `json:"status"`
}

func (s *ReservationService) CreateAdmin(companyID string, actor ActorContext, in CreateReservationInput) (*models.Reservation, error) {
	res, err := s.CreatePublic(companyID, in)
	if err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "reservation_created_admin", "reservation", res.ID, nil)
	return res, nil
}

type ListReservationsFilter struct {
	Date      *time.Time
	Time      *time.Time
	PartySize *int
	Status    models.ReservationStatus
}

func (s *ReservationService) List(companyID string, filter ListReservationsFilter) ([]models.Reservation, error) {
	q := s.db.Where("company_id = ?", companyID)
	if filter.Date != nil {
		start := time.Date(filter.Date.Year(), filter.Date.Month(), filter.Date.Day(), 0, 0, 0, 0, filter.Date.Location())
		q = q.Where("scheduled_at >= ? AND scheduled_at < ?", start, start.Add(24*time.Hour))
	}
	if filter.Time != nil {
		q = q.Where(
			"EXTRACT(HOUR FROM scheduled_at) = ? AND EXTRACT(MINUTE FROM scheduled_at) = ?",
			filter.Time.Hour(), filter.Time.Minute(),
		)
	}
	if filter.PartySize != nil {
		q = q.Where("party_size = ?", *filter.PartySize)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}

	var list []models.Reservation
	err := q.Order("scheduled_at ASC").Find(&list).Error
	return list, err
}

func (s *ReservationService) Get(companyID, id string) (*models.Reservation, error) {
	var res models.Reservation
	if err := s.db.Where("company_id = ? AND id = ?", companyID, id).First(&res).Error; err != nil {
		return nil, ErrNotFound
	}
	return &res, nil
}

func (s *ReservationService) Update(companyID string, actor ActorContext, id string, in UpdateReservationInput) (*models.Reservation, error) {
	res, err := s.Get(companyID, id)
	if err != nil {
		return nil, err
	}

	if in.GuestName != nil {
		res.GuestName = *in.GuestName
	}
	if in.GuestContact != nil {
		res.GuestContact = *in.GuestContact
	}
	if in.PartySize != nil {
		res.PartySize = *in.PartySize
	}
	if in.ScheduledAt != nil {
		company, err := getCompany(s.db, companyID)
		if err != nil {
			return nil, err
		}
		if err := validateSchedule(company, *in.ScheduledAt); err != nil {
			return nil, err
		}
		res.ScheduledAt = *in.ScheduledAt
	}
	if in.Status != nil {
		res.Status = *in.Status
	}

	if in.PartySize != nil || in.ScheduledAt != nil {
		company, err := getCompany(s.db, companyID)
		if err != nil {
			return nil, err
		}
		if err := s.checkCapacity(company, res.ScheduledAt, res.PartySize, res.ID); err != nil {
			return nil, err
		}
	}

	if err := s.db.Save(res).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "reservation_updated", "reservation", res.ID, nil)
	return res, nil
}

func (s *ReservationService) Delete(companyID string, actor ActorContext, id string) error {
	res, err := s.Get(companyID, id)
	if err != nil {
		return err
	}
	if err := s.db.Delete(res).Error; err != nil {
		return err
	}
	_ = s.events.Log(actor, "reservation_cancelled", "reservation", id, nil)
	return nil
}

func (s *ReservationService) CheckIn(companyID string, actor ActorContext, id string) (*models.Reservation, error) {
	res, err := s.Get(companyID, id)
	if err != nil {
		return nil, err
	}
	if res.Status != models.ReservationStatusPending {
		return nil, errors.New("só é possível dar baixa em reserva pendente")
	}
	res.Status = models.ReservationStatusCompleted
	if err := s.db.Save(res).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "reservation_checked_in", "reservation", res.ID, nil)
	return res, nil
}

func (s *ReservationService) CleanupExpired(company models.Company) (int64, error) {
	_, _, toleranceMinutes, err := s.bookingPolicy(company.ID)
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-time.Duration(toleranceMinutes) * time.Minute)
	result := s.db.Where(
		"company_id = ? AND status = ? AND scheduled_at < ?",
		company.ID,
		models.ReservationStatusPending,
		cutoff,
	).Delete(&models.Reservation{})

	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected > 0 {
		_ = s.events.Log(ActorContext{
			Type: models.ActorTypeSystem, CompanyID: company.ID,
		}, "reservations_cleaned", "company", company.ID, map[string]interface{}{
			"count": result.RowsAffected,
		})
	}
	return result.RowsAffected, nil
}

var ErrNoMatchingTables = errors.New("não há mais mesas com aquela quantidade de lugares")

func (s *ReservationService) bookingPolicy(companyID string) (allowLarger bool, holdMinutes, toleranceMinutes int, err error) {
	if def, ok := models.ConfigByKey(models.ConfigAllowLargerTables); ok {
		allowLarger = def.BoolDefault()
	}
	if def, ok := models.ConfigByKey(models.ConfigTableHoldMinutes); ok {
		holdMinutes = def.IntDefault()
	}
	if def, ok := models.ConfigByKey(models.ConfigReservationToleranceMinutes); ok {
		toleranceMinutes = def.IntDefault()
	}

	var configs []models.CompanyConfig
	if err := s.db.Where("company_id = ?", companyID).Find(&configs).Error; err != nil {
		return false, 0, 0, err
	}
	for _, cfg := range configs {
		switch cfg.Key {
		case models.ConfigAllowLargerTables:
			allowLarger = cfg.Value == "true"
		case models.ConfigTableHoldMinutes:
			if n, convErr := strconv.Atoi(cfg.Value); convErr == nil && n > 0 {
				holdMinutes = n
			}
		case models.ConfigReservationToleranceMinutes:
			if n, convErr := strconv.Atoi(cfg.Value); convErr == nil && n > 0 {
				toleranceMinutes = n
			}
		}
	}
	return allowLarger, holdMinutes, toleranceMinutes, nil
}

func (s *ReservationService) checkCapacity(company *models.Company, slot time.Time, partySize int, excludeID string) error {
	if partySize < 1 {
		return errors.New("quantidade de lugares inválida")
	}

	allowLarger, holdMinutes, _, err := s.bookingPolicy(company.ID)
	if err != nil {
		return err
	}
	hold := time.Duration(holdMinutes) * time.Minute

	tablesQ := s.db.Model(&models.Table{}).Where("company_id = ?", company.ID)
	if allowLarger {
		tablesQ = tablesQ.Where("capacity >= ?", partySize)
	} else {
		tablesQ = tablesQ.Where("capacity = ?", partySize)
	}
	var tableCount int64
	if err := tablesQ.Count(&tableCount).Error; err != nil {
		return err
	}
	if tableCount == 0 {
		return ErrNoMatchingTables
	}

	resQ := s.db.Model(&models.Reservation{}).Where(
		"company_id = ? AND party_size = ? AND status IN ? AND scheduled_at > ? AND scheduled_at <= ?",
		company.ID,
		partySize,
		[]models.ReservationStatus{models.ReservationStatusPending},
		slot.Add(-hold),
		slot,
	)
	if excludeID != "" {
		resQ = resQ.Where("id <> ?", excludeID)
	}
	var reservationCount int64
	if err := resQ.Count(&reservationCount).Error; err != nil {
		return err
	}
	if reservationCount >= tableCount {
		return ErrNoMatchingTables
	}
	return nil
}

func validateSchedule(company *models.Company, scheduled time.Time) error {
	if company.ReservationMode == models.ReservationModeFixed {
		fixed, err := parseTimeOnDate(company.FixedTime, scheduled)
		if err != nil {
			return errors.New("horário fixo inválido na configuração")
		}
		if !scheduled.Equal(fixed) {
			return errors.New("horário não permitido")
		}
		return nil
	}

	opens, err := parseTimeOnDate(company.OpensAt, scheduled)
	if err != nil {
		return err
	}
	closes, err := parseTimeOnDate(company.ClosesAt, scheduled)
	if err != nil {
		return err
	}
	if scheduled.Before(opens) || !scheduled.Before(closes) {
		return errors.New("fora do horário de funcionamento")
	}

	interval := time.Duration(company.SlotIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	diff := scheduled.Sub(opens)
	if diff%interval != 0 {
		return errors.New("horário não corresponde a um slot válido")
	}
	return nil
}

func buildSlots(company *models.Company, date time.Time) []time.Time {
	if company.ReservationMode == models.ReservationModeFixed {
		t, err := parseTimeOnDate(company.FixedTime, date)
		if err != nil {
			return nil
		}
		return []time.Time{t}
	}

	opens, _ := parseTimeOnDate(company.OpensAt, date)
	closes, _ := parseTimeOnDate(company.ClosesAt, date)
	interval := time.Duration(company.SlotIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 30 * time.Minute
	}

	var slots []time.Time
	for t := opens; t.Before(closes); t = t.Add(interval) {
		slots = append(slots, t)
	}
	return slots
}

func parseTimeOnDate(hhmm string, date time.Time) (time.Time, error) {
	parts := strings.Split(hhmm, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("formato inválido")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), h, m, 0, 0, date.Location()), nil
}
