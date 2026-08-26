package services

import (
	"reservation/api/internal/models"
)

func (s *TableService) GetByPublicToken(token string) (*models.Table, error) {
	var table models.Table
	if err := s.db.Where("public_token = ?", token).First(&table).Error; err != nil {
		return nil, ErrNotFound
	}
	return &table, nil
}

func (s *TableService) GetPublicWithMenu(token string) (map[string]interface{}, error) {
	table, err := s.GetByPublicToken(token)
	if err != nil {
		return nil, err
	}
	var menu []models.MenuItem
	if err := s.db.Where("company_id = ? AND active = ?", table.CompanyID, true).
		Order("name ASC").Find(&menu).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"table": table,
		"menu":  menu,
	}, nil
}
