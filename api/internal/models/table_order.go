package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TableOrderStatus string

const (
	TableOrderStatusPending TableOrderStatus = "pending"
	TableOrderStatusCompleted TableOrderStatus = "completed"
)

type TableOrder struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID string `gorm:"type:uuid;not null;index" json:"company_id"`
	TableSessionID string `gorm:"type:uuid;not null;index" json:"table_session_id"`
	TableID string `gorm:"type:uuid;not null;index" json:"table_id"`
	Status TableOrderStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Items []TableOrderItem `gorm:"foreignKey:TableOrderID" json:"items,omitempty"`
}

func (o *TableOrder) BeforeCreate(_ *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	return nil
}

type TableOrderItem struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`
	TableOrderID string `gorm:"type:uuid;not null;index" json:"table_order_id"`
	MenuItemID string `gorm:"type:uuid;not null" json:"menu_item_id"`
	MenuItemName string `gorm:"not null" json:"menu_item_name"`
	Quantity int `gorm:"not null" json:"quantity"`
	UnitPrice int64 `gorm:"not null" json:"unit_price_cents"`
	CreatedAt time.Time `json:"created_at"`
}

func (oi *TableOrderItem) BeforeCreate(_ *gorm.DB) error {
	if oi.ID == "" {
		oi.ID = uuid.NewString()
	}
	return nil
}
