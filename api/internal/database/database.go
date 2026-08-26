package database

import (
	"reservation/api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Company{},
		&models.CompanyConfig{},
		&models.User{},
		&models.Session{},
		&models.Table{},
		&models.MenuItem{},
		&models.Reservation{},
		&models.TableSession{},
		&models.TableOrder{},
		&models.TableOrderItem{},
		&models.PaymentHistory{},
		&models.WaitlistEntry{},
		&models.SystemEvent{},
	); err != nil {
		return err
	}
	// E-mail é único por companhia, não global. Remove índice antigo do GORM.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_users_email`).Error; err != nil {
		return err
	}
	if err := db.Exec(`ALTER TABLE users DROP CONSTRAINT IF EXISTS uni_users_email`).Error; err != nil {
		return err
	}
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_avulso ON users (email) WHERE company_id IS NULL`).Error
}
