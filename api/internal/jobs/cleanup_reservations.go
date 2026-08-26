package jobs

import (
	"log"

	"reservation/api/internal/models"
	"reservation/api/internal/services"

	"gorm.io/gorm"
)

type CleanupJob struct {
	db           *gorm.DB
	reservations *services.ReservationService
}

func NewCleanupJob(db *gorm.DB, reservations *services.ReservationService) *CleanupJob {
	return &CleanupJob{db: db, reservations: reservations}
}

func (j *CleanupJob) Run() {
	var companies []models.Company
	if err := j.db.Find(&companies).Error; err != nil {
		log.Printf("cleanup: erro ao listar empresas: %v", err)
		return
	}

	for _, company := range companies {
		count, err := j.reservations.CleanupExpired(company)
		if err != nil {
			log.Printf("cleanup company %s: %v", company.ID, err)
			continue
		}
		if count > 0 {
			log.Printf("cleanup company %s: %d reservas removidas", company.ID, count)
		}
	}
}
