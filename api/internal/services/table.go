package services

import (
	"encoding/base64"
	"errors"
	"fmt"

	"reservation/api/internal/config"
	"reservation/api/internal/models"
	"time"

	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type TableService struct {
	db     *gorm.DB
	cfg    config.Config
	events *SystemEventService
}

func NewTableService(db *gorm.DB, cfg config.Config, events *SystemEventService) *TableService {
	return &TableService{db: db, cfg: cfg, events: events}
}

type CreateTableInput struct {
	TableNumber string `json:"table_number"`
	Capacity    int    `json:"capacity"`
}

type UpdateTableInput struct {
	TableNumber *string `json:"table_number"`
	Capacity    *int    `json:"capacity"`
}

func (s *TableService) Create(companyID string, actor ActorContext, in CreateTableInput) (*models.Table, error) {
	table := models.Table{
		CompanyID:   companyID,
		TableNumber: in.TableNumber,
		Capacity:    in.Capacity,
		Status:      models.TableStatusAvailable,
	}
	if err := s.db.Create(&table).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "table_created", "table", table.ID, map[string]interface{}{
		"table_number": table.TableNumber,
	})
	return &table, nil
}

func (s *TableService) List(companyID string) ([]models.Table, error) {
	var tables []models.Table
	err := s.db.Where("company_id = ?", companyID).Order("table_number ASC").Find(&tables).Error
	return tables, err
}

func (s *TableService) Get(companyID, id string) (*models.Table, error) {
	var table models.Table
	if err := s.db.Where("company_id = ? AND id = ?", companyID, id).First(&table).Error; err != nil {
		return nil, ErrNotFound
	}
	return &table, nil
}

func (s *TableService) Update(companyID string, actor ActorContext, id string, in UpdateTableInput) (*models.Table, error) {
	table, err := s.Get(companyID, id)
	if err != nil {
		return nil, err
	}
	if in.TableNumber != nil {
		table.TableNumber = *in.TableNumber
	}
	if in.Capacity != nil {
		table.Capacity = *in.Capacity
	}
	if err := s.db.Save(table).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "table_updated", "table", table.ID, nil)
	return table, nil
}

func (s *TableService) Delete(companyID string, actor ActorContext, id string) error {
	table, err := s.Get(companyID, id)
	if err != nil {
		return err
	}
	if err := s.db.Delete(table).Error; err != nil {
		return err
	}
	_ = s.events.Log(actor, "table_deleted", "table", id, nil)
	return nil
}

func (s *TableService) SetStatus(companyID string, actor ActorContext, id string, status models.TableStatus) (*models.Table, error) {
	table, err := s.Get(companyID, id)
	if err != nil {
		return nil, err
	}

	if status != models.TableStatusAvailable && status != models.TableStatusOccupied {
		return nil, errors.New("status inválido")
	}

	if status == models.TableStatusOccupied && table.Status == models.TableStatusAvailable {
		session := models.TableSession{
			CompanyID: companyID,
			TableID:   table.ID,
			OpenedAt:  time.Now(),
		}
		if err := s.db.Create(&session).Error; err != nil {
			return nil, err
		}
		_ = s.events.Log(actor, "table_occupied", "table", table.ID, map[string]interface{}{
			"session_id": session.ID,
		})
	}

	if status == models.TableStatusAvailable && table.Status == models.TableStatusOccupied {
		var openSession models.TableSession
		if err := s.db.Where("table_id = ? AND closed_at IS NULL", table.ID).First(&openSession).Error; err == nil {
			var count int64
			s.db.Model(&models.TableOrder{}).Where("table_session_id = ? AND status = ?", openSession.ID, models.TableOrderStatusPending).Count(&count)
			if count > 0 {
				return nil, errors.New("mesa possui pedidos em aberto")
			}
		}
		_ = s.events.Log(actor, "table_freed", "table", table.ID, nil)
	}

	table.Status = status
	if err := s.db.Save(table).Error; err != nil {
		return nil, err
	}
	return table, nil
}

func (s *TableService) QRCode(companyID, id string) (string, string, error) {
	table, err := s.Get(companyID, id)
	if err != nil {
		return "", "", err
	}
	url := fmt.Sprintf("%s/public/tables/%s", s.cfg.AppPublicURL, table.PublicToken)
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return "", "", err
	}
	b64 := base64.StdEncoding.EncodeToString(png)
	return url, b64, nil
}
