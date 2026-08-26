package services

import (
	"encoding/json"
	"errors"
	"time"

	"reservation/api/internal/models"

	"gorm.io/gorm"
)

type PaymentService struct {
	db     *gorm.DB
	events *SystemEventService
}

func NewPaymentService(db *gorm.DB, events *SystemEventService) *PaymentService {
	return &PaymentService{db: db, events: events}
}

func (s *PaymentService) PayTable(companyID string, actor ActorContext, tableID string) (*models.PaymentHistory, error) {
	var table models.Table
	if err := s.db.Where("company_id = ? AND id = ?", companyID, tableID).First(&table).Error; err != nil {
		return nil, ErrNotFound
	}

	var session models.TableSession
	if err := s.db.Where("table_id = ? AND closed_at IS NULL", tableID).First(&session).Error; err != nil {
		return nil, errors.New("mesa sem sessão aberta")
	}

	var list []models.TableOrder
	if err := s.db.Preload("Items").Where("table_session_id = ?", session.ID).Find(&list).Error; err != nil {
		return nil, err
	}

	total := int64(0)
	type itemSnap struct {
		Name     string `json:"name"`
		Quantity int    `json:"quantity"`
		Price    int64  `json:"price_cents"`
	}
	snapshot := make([]itemSnap, 0)
	for _, o := range list {
		for _, item := range o.Items {
			total += item.UnitPrice * int64(item.Quantity)
			snapshot = append(snapshot, itemSnap{
				Name: item.MenuItemName, Quantity: item.Quantity, Price: item.UnitPrice,
			})
		}
	}

	snapJSON, _ := json.Marshal(snapshot)
	now := time.Now()
	history := models.PaymentHistory{
		CompanyID:      companyID,
		TableID:        table.ID,
		TableNumber:    table.TableNumber,
		TableSessionID: session.ID,
		TotalCents:     total,
		ItemsSnapshot:  string(snapJSON),
		PaidAt:         now,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&history).Error; err != nil {
			return err
		}
		session.ClosedAt = &now
		if err := tx.Save(&session).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.TableOrder{}).Where("table_session_id = ?", session.ID).
			Update("status", models.TableOrderStatusCompleted).Error; err != nil {
			return err
		}
		table.Status = models.TableStatusAvailable
		return tx.Save(&table).Error
	})
	if err != nil {
		return nil, err
	}

	_ = s.events.Log(actor, "payment_completed", "payment_history", history.ID, map[string]interface{}{
		"table_id":    tableID,
		"total_cents": total,
	})

	return &history, nil
}

func (s *PaymentService) ListHistories(companyID string) ([]models.PaymentHistory, error) {
	var list []models.PaymentHistory
	err := s.db.Where("company_id = ?", companyID).Order("paid_at DESC").Find(&list).Error
	return list, err
}
