package services

import (
	"errors"
	"strings"

	"reservation/api/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrEmailTakenInCompany = errors.New("já existe um usuário com este e-mail nesta companhia")

type UserService struct {
	db     *gorm.DB
	events *SystemEventService
}

func NewUserService(db *gorm.DB, events *SystemEventService) *UserService {
	return &UserService{db: db, events: events}
}

type UserInput struct {
	Name     string          `json:"name"`
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Role     models.UserRole `json:"role"`
}

func (s *UserService) Create(companyID string, actor ActorContext, in UserInput) (*models.User, error) {
	if in.Role == models.RoleDev && actor.Type == models.ActorTypeUser {
		var actorUser models.User
		if err := s.db.First(&actorUser, "id = ?", *actor.UserID).Error; err != nil || actorUser.Role != models.RoleDev {
			return nil, errors.New("apenas dev pode criar usuário dev")
		}
	}

	email := normalizeEmail(in.Email)
	taken, err := emailExistsInCompany(s.db, companyID, email, "")
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrEmailTakenInCompany
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		CompanyID:    companyIDPtr(companyID),
		Name:         in.Name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         in.Role,
	}
	if user.Role == "" {
		user.Role = models.RoleCashier
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	_ = s.events.Log(actor, "user_created", "user", user.ID, map[string]interface{}{
		"email": user.Email,
		"role":  user.Role,
	})
	return &user, nil
}

func (s *UserService) List(companyID string) ([]models.User, error) {
	var users []models.User
	err := s.db.Where("company_id = ?", companyID).Order("created_at DESC").Find(&users).Error
	return users, err
}

func (s *UserService) Get(companyID, id string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("company_id = ? AND id = ?", companyID, id).First(&user).Error; err != nil {
		return nil, ErrNotFound
	}
	return &user, nil
}

func (s *UserService) Update(companyID string, actor ActorContext, id string, in UserInput) (*models.User, error) {
	user, err := s.Get(companyID, id)
	if err != nil {
		return nil, err
	}

	if in.Name != "" {
		user.Name = in.Name
	}
	if in.Email != "" {
		email := normalizeEmail(in.Email)
		taken, err := emailExistsInCompany(s.db, companyID, email, user.ID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrEmailTakenInCompany
		}
		user.Email = email
	}
	if in.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}
	if in.Role != "" {
		if in.Role == models.RoleDev {
			var actorUser models.User
			if err := s.db.First(&actorUser, "id = ?", *actor.UserID).Error; err != nil || actorUser.Role != models.RoleDev {
				return nil, errors.New("apenas dev pode atribuir role dev")
			}
		}
		user.Role = in.Role
	}

	if err := s.db.Save(user).Error; err != nil {
		return nil, err
	}
	_ = s.events.Log(actor, "user_updated", "user", user.ID, nil)
	return user, nil
}

func (s *UserService) Delete(companyID string, actor ActorContext, id string) error {
	user, err := s.Get(companyID, id)
	if err != nil {
		return err
	}
	if err := s.db.Delete(user).Error; err != nil {
		return err
	}
	_ = s.events.Log(actor, "user_deleted", "user", id, nil)
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func emailExistsInCompany(db *gorm.DB, companyID, email, excludeUserID string) (bool, error) {
	q := db.Model(&models.User{}).Where("email = ?", email)
	if companyID == "" {
		q = q.Where("company_id IS NULL")
	} else {
		q = q.Where("company_id = ?", companyID)
	}
	if excludeUserID != "" {
		q = q.Where("id <> ?", excludeUserID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func companyIDPtr(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
