package services

import (
	"errors"
	"strconv"

	"reservation/api/internal/models"

	"gorm.io/gorm"
)

type CompanyConfigView struct {
	Key   string      `json:"key"`
	Label string      `json:"label"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type UpdateCompanyConfigInput struct {
	Value interface{} `json:"value"`
}

func seedCompanyConfigs(db *gorm.DB, companyID string) error {
	for _, def := range models.ConfigCatalog {
		cfg := models.CompanyConfig{
			CompanyID: companyID,
			Key:       def.Key,
			Value:     def.Default,
		}
		if err := db.Where("company_id = ? AND key = ?", companyID, def.Key).
			FirstOrCreate(&cfg).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *CompanyService) ListConfigs(companyID string) ([]CompanyConfigView, error) {
	if err := seedCompanyConfigs(s.db, companyID); err != nil {
		return nil, err
	}
	var rows []models.CompanyConfig
	if err := s.db.Where("company_id = ?", companyID).Find(&rows).Error; err != nil {
		return nil, err
	}
	byKey := make(map[string]string, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row.Value
	}

	out := make([]CompanyConfigView, 0, len(models.ConfigCatalog))
	for _, def := range models.ConfigCatalog {
		val := def.Default
		if stored, ok := byKey[def.Key]; ok {
			val = stored
		}
		parsed, err := parseConfigValue(def.Type, val)
		if err != nil {
			return nil, err
		}
		out = append(out, CompanyConfigView{
			Key:   def.Key,
			Label: def.Label,
			Type:  def.Type,
			Value: parsed,
		})
	}
	return out, nil
}

func (s *CompanyService) UpdateConfig(companyID, key string, raw interface{}) ([]CompanyConfigView, error) {
	def, ok := models.ConfigByKey(key)
	if !ok {
		return nil, errors.New("configuração desconhecida")
	}
	stored, err := stringifyConfigValue(def, raw)
	if err != nil {
		return nil, err
	}
	if err := seedCompanyConfigs(s.db, companyID); err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.CompanyConfig{}).
		Where("company_id = ? AND key = ?", companyID, key).
		Update("value", stored).Error; err != nil {
		return nil, err
	}
	return s.ListConfigs(companyID)
}

func parseConfigValue(cfgType, raw string) (interface{}, error) {
	switch cfgType {
	case models.ConfigTypeBoolean:
		return raw == "true", nil
	case models.ConfigTypeInteger:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, errors.New("valor da configuração inválido")
		}
		return n, nil
	default:
		return raw, nil
	}
}

func stringifyConfigValue(def models.ConfigDef, raw interface{}) (string, error) {
	switch def.Type {
	case models.ConfigTypeBoolean:
		b, ok := raw.(bool)
		if !ok {
			return "", errors.New("value deve ser true ou false")
		}
		if b {
			return "true", nil
		}
		return "false", nil
	case models.ConfigTypeInteger:
		n, ok := jsonNumber(raw)
		if !ok || n < 1 || n > 1440 {
			return "", errors.New("value deve ser um número de minutos entre 1 e 1440")
		}
		return strconv.Itoa(n), nil
	default:
		return "", errors.New("tipo de configuração inválido")
	}
}

func jsonNumber(raw interface{}) (int, bool) {
	switch v := raw.(type) {
	case float64:
		n := int(v)
		if float64(n) != v {
			return 0, false
		}
		return n, true
	case int:
		return v, true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}
