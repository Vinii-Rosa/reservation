package services

import (
	"errors"
	"time"

	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const sessionDuration = 30 * 24 * time.Hour

type AuthService struct {
	db        *gorm.DB
	jwtSecret []byte
	events    *SystemEventService
}

func NewAuthService(db *gorm.DB, jwtSecret string, events *SystemEventService) *AuthService {
	return &AuthService{db: db, jwtSecret: []byte(jwtSecret), events: events}
}

type RegisterInput struct {
	Name     string          `json:"name"`
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Role     models.UserRole `json:"role"`
}

type LoginInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	CompanyID string `json:"company_id"`
}

type AuthResponse struct {
	Token   string           `json:"token"`
	Expires time.Time        `json:"expires_at"`
	User    models.User      `json:"user"`
	Company *models.Company  `json:"company,omitempty"`
}

func (s *AuthService) Register(in RegisterInput) (*AuthResponse, error) {
	if in.Name == "" || in.Email == "" || in.Password == "" {
		return nil, errors.New("dados obrigatórios ausentes")
	}
	if in.Role == "" {
		return nil, errors.New("role é obrigatória")
	}
	if in.Role != models.RoleAdmin && in.Role != models.RoleCashier {
		return nil, errors.New("role inválida; use admin ou cashier")
	}

	email := normalizeEmail(in.Email)
	taken, err := emailExistsInCompany(s.db, "", email, "")
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, errors.New("já existe um usuário com este e-mail")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Name:         in.Name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         in.Role,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	token, expires, err := s.createSession(s.db, user)
	if err != nil {
		return nil, err
	}

	_ = s.events.Log(ActorContext{
		Type: models.ActorTypeUser, UserID: &user.ID, Name: user.Name,
	}, "user_registered", "user", user.ID, map[string]interface{}{"role": user.Role})

	return &AuthResponse{Token: token, Expires: expires, User: user}, nil
}

func (s *AuthService) Login(in LoginInput) (*AuthResponse, error) {
	email := normalizeEmail(in.Email)
	q := s.db.Where("email = ?", email)
	if in.CompanyID != "" {
		q = q.Where("company_id = ?", in.CompanyID)
	} else {
		var count int64
		if err := s.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
			return nil, errors.New("credenciais inválidas")
		}
		if count > 1 {
			return nil, errors.New("e-mail cadastrado em mais de uma companhia; informe company_id")
		}
	}

	var users []models.User
	if err := q.Find(&users).Error; err != nil {
		return nil, errors.New("credenciais inválidas")
	}
	if len(users) == 0 {
		return nil, errors.New("credenciais inválidas")
	}
	if len(users) > 1 {
		return nil, errors.New("e-mail cadastrado em mais de uma companhia; informe company_id")
	}

	user := users[0]
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		return nil, errors.New("credenciais inválidas")
	}

	var company *models.Company
	if cid := user.CompanyIDValue(); cid != "" {
		var c models.Company
		if err := s.db.First(&c, "id = ?", cid).Error; err != nil {
			return nil, err
		}
		company = &c
	}

	token, expires, err := s.createSession(s.db, user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, Expires: expires, User: user, Company: company}, nil
}

func (s *AuthService) Logout(token string) error {
	hash := middleware.HashToken(token)
	return s.db.Where("token_hash = ?", hash).Delete(&models.Session{}).Error
}

func (s *AuthService) createSession(db *gorm.DB, user models.User) (string, time.Time, error) {
	expires := time.Now().Add(sessionDuration)
	claims := middleware.AuthClaims{
		UserID:    user.ID,
		CompanyID: user.CompanyIDValue(),
		Role:      user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	session := models.Session{
		UserID:    user.ID,
		TokenHash: middleware.HashToken(signed),
		ExpiresAt: expires,
	}
	if err := db.Create(&session).Error; err != nil {
		return "", time.Time{}, err
	}
	return signed, expires, nil
}
