package controllers

import (
	"net/http"
	"strings"

	"reservation/api/internal/middleware"
	"reservation/api/internal/models"
	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	auth *services.AuthService
}

func NewAuthController(auth *services.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

func (c *AuthController) Register(ctx *gin.Context) {
	var in services.RegisterInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	res, err := c.auth.Register(in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, res)
}

func (c *AuthController) Login(ctx *gin.Context) {
	var in services.LoginInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	res, err := c.auth.Login(in)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, res)
}

func (c *AuthController) Logout(ctx *gin.Context) {
	header := ctx.GetHeader("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "token ausente"})
		return
	}
	if err := c.auth.Logout(strings.TrimSpace(parts[1])); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "logout realizado"})
}

func userFromContext(ctx *gin.Context) services.ActorContext {
	userID, _ := ctx.Get(middleware.ContextUserID)
	companyID, _ := ctx.Get(middleware.ContextCompanyID)
	name, _ := ctx.Get(middleware.ContextUserName)
	uid := userID.(string)
	return services.ActorContext{
		Type:      models.ActorTypeUser,
		UserID:    &uid,
		Name:      name.(string),
		CompanyID: companyID.(string),
	}
}

func companyFromContext(ctx *gin.Context) string {
	companyID, _ := ctx.Get(middleware.ContextCompanyID)
	return companyID.(string)
}
