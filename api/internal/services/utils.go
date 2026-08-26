package services

import (
	"reservation/api/internal/models"

	"gorm.io/gorm"
)

func getCompany(db *gorm.DB, id string) (*models.Company, error) {
	var company models.Company
	if err := db.First(&company, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &company, nil
}

func getCompanyByWaitlistToken(db *gorm.DB, token string) (*models.Company, error) {
	var company models.Company
	if err := db.Where("waitlist_token = ?", token).First(&company).Error; err != nil {
		return nil, ErrNotFound
	}
	return &company, nil
}
