package services

import (
	"errors"
	"time"

	"reservation/api/internal/models"
)

func (s *TableOrderService) CreatePublic(tableToken, guestName string, in CreateTableOrderInput) (*models.TableOrder, error) {
	var table models.Table
	if err := s.db.Where("public_token = ?", tableToken).First(&table).Error; err != nil {
		return nil, ErrNotFound
	}

	if table.Status == models.TableStatusAvailable {
		table.Status = models.TableStatusOccupied
		if err := s.db.Save(&table).Error; err != nil {
			return nil, err
		}
		session := models.TableSession{
			CompanyID: table.CompanyID,
			TableID:   table.ID,
			OpenedAt:  time.Now(),
		}
		if err := s.db.Create(&session).Error; err != nil {
			return nil, err
		}
	} else if table.Status != models.TableStatusOccupied {
		return nil, errors.New("mesa indisponível")
	}

	session, err := s.getOpenSession(table.ID)
	if err != nil {
		return nil, err
	}

	tableOrder, err := s.create(table.CompanyID, table.ID, session.ID, in)
	if err != nil {
		return nil, err
	}

	_ = s.events.Log(ActorContext{
		Type: models.ActorTypeClient, Name: guestName, CompanyID: table.CompanyID,
	}, "table_order_created", "table_order", tableOrder.ID, map[string]interface{}{
		"table_id": table.ID,
	})

	return tableOrder, nil
}
