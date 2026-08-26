package services

import (
	"errors"
	"strings"
	"unicode"

	"reservation/api/internal/models"

	"gorm.io/gorm"
)

var (
	ErrCompanyDocumentTaken = errors.New("já existe uma companhia com este CPF/CNPJ")
	ErrCompanyEmailTaken    = errors.New("já existe uma companhia com este e-mail")
	ErrUserAlreadyInCompany = errors.New("usuário já está vinculado a uma companhia")
)

type CompanyService struct {
	db     *gorm.DB
	events *SystemEventService
}

func NewCompanyService(db *gorm.DB, events *SystemEventService) *CompanyService {
	return &CompanyService{db: db, events: events}
}

type CreateCompanyInput struct {
	Name            string              `json:"name"`
	DocumentType    models.DocumentType `json:"document_type"`
	Document        string              `json:"document"`
	Email           string              `json:"email"`
	Phone           string              `json:"phone"`
	Address         models.Address      `json:"address"`
	ProfilePhotoURL string              `json:"profile_photo_url"`
}

type UpdateCompanyInput struct {
	Name            *string              `json:"name"`
	DocumentType    *models.DocumentType `json:"document_type"`
	Document        *string              `json:"document"`
	Email           *string              `json:"email"`
	Phone           *string              `json:"phone"`
	Address         *models.Address      `json:"address"`
	ProfilePhotoURL *string              `json:"profile_photo_url"`
}

type UpdateScheduleInput struct {
	ReservationMode     *models.ReservationMode `json:"reservation_mode"`
	OpensAt             *string                 `json:"opens_at"`
	ClosesAt            *string                 `json:"closes_at"`
	FixedTime           *string                 `json:"fixed_time"`
	SlotIntervalMinutes *int                    `json:"slot_interval_minutes"`
	AvgTurnoverMinutes  *int                    `json:"avg_turnover_minutes"`
}

func (s *CompanyService) Create(actor ActorContext, in CreateCompanyInput) (*models.Company, error) {
	if actor.UserID == nil {
		return nil, errors.New("usuário ausente")
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", *actor.UserID).Error; err != nil {
		return nil, ErrNotFound
	}
	if user.CompanyIDValue() != "" {
		return nil, ErrUserAlreadyInCompany
	}

	company, err := buildCompany(in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUnique(company.Document, company.Email, ""); err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(company).Error; err != nil {
			return err
		}
		if err := tx.Model(&user).Update("company_id", company.ID).Error; err != nil {
			return err
		}
		return seedCompanyConfigs(tx, company.ID)
	})
	if err != nil {
		return nil, err
	}

	_ = s.events.Log(ActorContext{
		Type: models.ActorTypeUser, UserID: actor.UserID, Name: actor.Name, CompanyID: company.ID,
	}, "company_created", "company", company.ID, nil)
	return company, nil
}

func (s *CompanyService) Get(companyID string) (*models.Company, error) {
	return getCompany(s.db, companyID)
}

func (s *CompanyService) Update(companyID string, actor ActorContext, in UpdateCompanyInput) (*models.Company, error) {
	company, err := s.Get(companyID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		company.Name = strings.TrimSpace(*in.Name)
	}
	if in.DocumentType != nil {
		company.DocumentType = *in.DocumentType
	}
	if in.Document != nil {
		company.Document = digitsOnly(*in.Document)
	}
	if in.Email != nil {
		company.Email = normalizeEmail(*in.Email)
	}
	if in.Phone != nil {
		company.Phone = strings.TrimSpace(*in.Phone)
	}
	if in.Address != nil {
		company.Address = normalizeAddress(*in.Address)
	}
	if in.ProfilePhotoURL != nil {
		company.ProfilePhotoURL = strings.TrimSpace(*in.ProfilePhotoURL)
	}

	if err := validateCompanyFields(company); err != nil {
		return nil, err
	}
	if err := s.ensureUnique(company.Document, company.Email, company.ID); err != nil {
		return nil, err
	}
	if err := s.db.Save(company).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "company_updated", "company", company.ID, nil)
	return company, nil
}

func (s *CompanyService) UpdateSchedule(companyID string, in UpdateScheduleInput) (*models.Company, error) {
	company, err := s.Get(companyID)
	if err != nil {
		return nil, err
	}

	if in.ReservationMode != nil {
		if *in.ReservationMode != models.ReservationModeRange && *in.ReservationMode != models.ReservationModeFixed {
			return nil, errors.New("reservation_mode inválido")
		}
		company.ReservationMode = *in.ReservationMode
	}
	if in.OpensAt != nil {
		company.OpensAt = *in.OpensAt
	}
	if in.ClosesAt != nil {
		company.ClosesAt = *in.ClosesAt
	}
	if in.FixedTime != nil {
		company.FixedTime = *in.FixedTime
	}
	if in.SlotIntervalMinutes != nil {
		company.SlotIntervalMinutes = *in.SlotIntervalMinutes
	}
	if in.AvgTurnoverMinutes != nil {
		company.AvgTurnoverMinutes = *in.AvgTurnoverMinutes
	}

	if err := s.db.Save(company).Error; err != nil {
		return nil, err
	}
	return company, nil
}

func (s *CompanyService) GetByWaitlistToken(token string) (*models.Company, error) {
	return getCompanyByWaitlistToken(s.db, token)
}

func (s *CompanyService) ensureUnique(document, email, excludeID string) error {
	qDoc := s.db.Model(&models.Company{}).Where("document = ?", document)
	qEmail := s.db.Model(&models.Company{}).Where("email = ?", email)
	if excludeID != "" {
		qDoc = qDoc.Where("id <> ?", excludeID)
		qEmail = qEmail.Where("id <> ?", excludeID)
	}
	var n int64
	if err := qDoc.Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrCompanyDocumentTaken
	}
	if err := qEmail.Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrCompanyEmailTaken
	}
	return nil
}

func buildCompany(in CreateCompanyInput) (*models.Company, error) {
	company := &models.Company{
		Name:            strings.TrimSpace(in.Name),
		DocumentType:    in.DocumentType,
		Document:        digitsOnly(in.Document),
		Email:           normalizeEmail(in.Email),
		Phone:           strings.TrimSpace(in.Phone),
		Address:         normalizeAddress(in.Address),
		ProfilePhotoURL: strings.TrimSpace(in.ProfilePhotoURL),
	}
	if err := validateCompanyFields(company); err != nil {
		return nil, err
	}
	return company, nil
}

func validateCompanyFields(c *models.Company) error {
	if c.Name == "" || c.Email == "" || c.Phone == "" {
		return errors.New("nome, e-mail e telefone são obrigatórios")
	}
	if c.DocumentType != models.DocumentCNPJ && c.DocumentType != models.DocumentCPF {
		return errors.New("document_type deve ser cnpj ou cpf")
	}
	if c.DocumentType == models.DocumentCPF && len(c.Document) != 11 {
		return errors.New("CPF deve ter 11 dígitos")
	}
	if c.DocumentType == models.DocumentCNPJ && len(c.Document) != 14 {
		return errors.New("CNPJ deve ter 14 dígitos")
	}
	if c.Address.ZipCode == "" || c.Address.Street == "" || c.Address.Number == "" ||
		c.Address.City == "" || c.Address.State == "" {
		return errors.New("endereço incompleto")
	}
	if len(c.Address.State) != 2 {
		return errors.New("UF inválida")
	}
	return nil
}

func normalizeAddress(a models.Address) models.Address {
	a.ZipCode = digitsOnly(a.ZipCode)
	a.Street = strings.TrimSpace(a.Street)
	a.Number = strings.TrimSpace(a.Number)
	a.Complement = strings.TrimSpace(a.Complement)
	a.Neighborhood = strings.TrimSpace(a.Neighborhood)
	a.City = strings.TrimSpace(a.City)
	a.State = strings.ToUpper(strings.TrimSpace(a.State))
	return a
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
