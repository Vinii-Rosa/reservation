package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleDev     UserRole = "dev"
	RoleAdmin   UserRole = "admin"
	RoleCashier UserRole = "cashier"
)

type User struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID    *string   `gorm:"type:uuid;uniqueIndex:idx_users_company_email" json:"company_id,omitempty"`
	Name         string    `gorm:"not null" json:"name"`
	Email        string    `gorm:"not null;uniqueIndex:idx_users_company_email" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         UserRole  `gorm:"type:varchar(20);not null;default:'cashier'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) CompanyIDValue() string {
	if u.CompanyID == nil {
		return ""
	}
	return *u.CompanyID
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
