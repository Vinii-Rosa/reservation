package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthClaims struct {
	UserID    string          `json:"user_id"`
	CompanyID string          `json:"company_id"`
	Role      models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type AuthMiddleware struct {
	db        *gorm.DB
	jwtSecret []byte
}

func NewAuthMiddleware(db *gorm.DB, jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{db: db, jwtSecret: []byte(jwtSecret)}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractBearer(c.GetHeader("Authorization"))
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token ausente"})
			return
		}

		claims := &AuthClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (interface{}, error) {
			return m.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
			return
		}

		hash := hashToken(tokenStr)
		var session models.Session
		if err := m.db.Where("token_hash = ? AND expires_at > ?", hash, time.Now()).First(&session).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "sessão inválida ou expirada"})
			return
		}

		var user models.User
		if err := m.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não encontrado"})
			return
		}

		c.Set(ContextUserID, user.ID)
		c.Set(ContextCompanyID, user.CompanyIDValue())
		c.Set(ContextRole, string(user.Role))
		c.Set(ContextUserName, user.Name)
		c.Next()
	}
}

func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[string(r)] = struct{}{}
	}

	return func(c *gin.Context) {
		role, ok := c.Get(ContextRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acesso negado"})
			return
		}

		roleStr := role.(string)
		if roleStr == string(models.RoleDev) {
			c.Next()
			return
		}

		if _, ok := allowed[roleStr]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permissão insuficiente"})
			return
		}
		c.Next()
	}
}

func RequireCompany() gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID, _ := c.Get(ContextCompanyID)
		id, _ := companyID.(string)
		if id == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "usuário sem companhia"})
			return
		}
		c.Next()
	}
}

func extractBearer(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func HashToken(token string) string {
	return hashToken(token)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
