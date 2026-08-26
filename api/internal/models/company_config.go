package models

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ConfigAllowLargerTables           = "allow_larger_tables"
	ConfigTableHoldMinutes            = "table_hold_minutes"
	ConfigReservationToleranceMinutes = "reservation_tolerance_minutes"

	ConfigTypeBoolean = "boolean"
	ConfigTypeInteger = "integer"
)

type ConfigDef struct {
	Key     string
	Type    string
	Label   string
	Default string
}

var ConfigCatalog = []ConfigDef{
	{
		Key:     ConfigAllowLargerTables,
		Type:    ConfigTypeBoolean,
		Label:   "Permitir mesa com mais lugares que o grupo",
		Default: "false",
	},
	{
		Key:     ConfigTableHoldMinutes,
		Type:    ConfigTypeInteger,
		Label:   "Tempo que a mesa fica presa após o horário da reserva (minutos)",
		Default: "120",
	},
	{
		Key:     ConfigReservationToleranceMinutes,
		Type:    ConfigTypeInteger,
		Label:   "Tolerância para o cliente chegar após o horário da reserva (minutos)",
		Default: "15",
	},
}

func ConfigByKey(key string) (ConfigDef, bool) {
	for _, def := range ConfigCatalog {
		if def.Key == key {
			return def, true
		}
	}
	return ConfigDef{}, false
}

func (d ConfigDef) BoolDefault() bool {
	return d.Default == "true"
}

func (d ConfigDef) IntDefault() int {
	n, err := strconv.Atoi(d.Default)
	if err != nil {
		return 0
	}
	return n
}

type CompanyConfig struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID string    `gorm:"type:uuid;not null;uniqueIndex:idx_company_configs_key" json:"company_id"`
	Key       string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_company_configs_key" json:"key"`
	Value     string    `gorm:"type:varchar(64);not null" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *CompanyConfig) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}
