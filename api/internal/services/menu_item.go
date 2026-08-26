package services

import (
	"reservation/api/internal/models"

	"gorm.io/gorm"
)

type MenuItemService struct {
	db     *gorm.DB
	events *SystemEventService
}

func NewMenuItemService(db *gorm.DB, events *SystemEventService) *MenuItemService {
	return &MenuItemService{db: db, events: events}
}

type CreateMenuItemInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Active      bool   `json:"active"`
}

type UpdateMenuItemInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	PriceCents  *int64  `json:"price_cents"`
	Active      *bool   `json:"active"`
}

func (s *MenuItemService) Create(companyID string, actor ActorContext, in CreateMenuItemInput) (*models.MenuItem, error) {
	item := models.MenuItem{
		CompanyID:   companyID,
		Name:        in.Name,
		Description: in.Description,
		PriceCents:  in.PriceCents,
		Active:      in.Active,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "menu_item_created", "menu_item", item.ID, nil)
	return &item, nil
}

func (s *MenuItemService) List(companyID string, activeOnly bool) ([]models.MenuItem, error) {
	q := s.db.Where("company_id = ?", companyID)
	if activeOnly {
		q = q.Where("active = ?", true)
	}
	var items []models.MenuItem
	err := q.Order("name ASC").Find(&items).Error
	return items, err
}

func (s *MenuItemService) Get(companyID, id string) (*models.MenuItem, error) {
	var item models.MenuItem
	if err := s.db.Where("company_id = ? AND id = ?", companyID, id).First(&item).Error; err != nil {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (s *MenuItemService) Update(companyID string, actor ActorContext, id string, in UpdateMenuItemInput) (*models.MenuItem, error) {
	item, err := s.Get(companyID, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		item.Name = *in.Name
	}
	if in.Description != nil {
		item.Description = *in.Description
	}
	if in.PriceCents != nil {
		item.PriceCents = *in.PriceCents
	}
	if in.Active != nil {
		item.Active = *in.Active
	}
	if err := s.db.Save(item).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "menu_item_updated", "menu_item", item.ID, nil)
	return item, nil
}

func (s *MenuItemService) Delete(companyID string, actor ActorContext, id string) error {
	item, err := s.Get(companyID, id)
	if err != nil {
		return err
	}
	if err := s.db.Delete(item).Error; err != nil {
		return err
	}
	_ = s.events.Log(actor, "menu_item_deleted", "menu_item", id, nil)
	return nil
}
