package services

import (
	"errors"

	"reservation/api/internal/models"

	"gorm.io/gorm"
)

type TableOrderService struct {
	db     *gorm.DB
	events *SystemEventService
}

func NewTableOrderService(db *gorm.DB, events *SystemEventService) *TableOrderService {
	return &TableOrderService{db: db, events: events}
}

type TableOrderItemInput struct {
	MenuItemID string `json:"menu_item_id"`
	Quantity   int    `json:"quantity"`
}

type CreateTableOrderInput struct {
	Items []TableOrderItemInput `json:"items"`
}

type TableOrderWithTable struct {
	models.TableOrder
	TableNumber string `json:"table_number"`
}

func (s *TableOrderService) create(companyID, tableID, sessionID string, in CreateTableOrderInput) (*models.TableOrder, error) {
	if len(in.Items) == 0 {
		return nil, errors.New("pedido vazio")
	}

	tableOrder := models.TableOrder{
		CompanyID:      companyID,
		TableID:        tableID,
		TableSessionID: sessionID,
		Status:         models.TableOrderStatusPending,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tableOrder).Error; err != nil {
			return err
		}
		for _, item := range in.Items {
			var menu models.MenuItem
			if err := tx.Where("company_id = ? AND id = ? AND active = ?", companyID, item.MenuItemID, true).First(&menu).Error; err != nil {
				return errors.New("item do menu inválido")
			}
			if item.Quantity < 1 {
				return errors.New("quantidade inválida")
			}
			line := models.TableOrderItem{
				TableOrderID: tableOrder.ID,
				MenuItemID:   menu.ID,
				MenuItemName: menu.Name,
				Quantity:     item.Quantity,
				UnitPrice:    menu.PriceCents,
			}
			if err := tx.Create(&line).Error; err != nil {
				return err
			}
			tableOrder.Items = append(tableOrder.Items, line)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &tableOrder, nil
}

func (s *TableOrderService) ListPendingTableOrders(companyID string) ([]TableOrderWithTable, error) {
	var list []models.TableOrder
	if err := s.db.Preload("Items").Where("company_id = ? AND status = ?", companyID, models.TableOrderStatusPending).
		Order("created_at ASC").Find(&list).Error; err != nil {
		return nil, err
	}

	result := make([]TableOrderWithTable, 0, len(list))
	for _, o := range list {
		var table models.Table
		_ = s.db.First(&table, "id = ?", o.TableID)
		result = append(result, TableOrderWithTable{TableOrder: o, TableNumber: table.TableNumber})
	}
	return result, nil
}

func (s *TableOrderService) GetTableOrderSummary(companyID, tableID string) (map[string]interface{}, error) {
	var table models.Table
	if err := s.db.Where("company_id = ? AND id = ?", companyID, tableID).First(&table).Error; err != nil {
		return nil, ErrNotFound
	}

	session, err := s.getOpenSession(table.ID)
	if err != nil {
		return map[string]interface{}{
			"table":        table,
			"session":      nil,
			"table_orders": []models.TableOrder{},
			"total_cents":  int64(0),
		}, nil
	}

	var list []models.TableOrder
	s.db.Preload("Items").Where("table_session_id = ?", session.ID).Find(&list)

	total := int64(0)
	for _, o := range list {
		for _, item := range o.Items {
			total += item.UnitPrice * int64(item.Quantity)
		}
	}

	return map[string]interface{}{
		"table":        table,
		"session":      session,
		"table_orders": list,
		"total_cents":  total,
	}, nil
}

func (s *TableOrderService) getOpenSession(tableID string) (*models.TableSession, error) {
	var session models.TableSession
	if err := s.db.Where("table_id = ? AND closed_at IS NULL", tableID).First(&session).Error; err != nil {
		return nil, errors.New("sessão da mesa não encontrada")
	}
	return &session, nil
}
