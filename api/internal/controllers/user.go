package controllers

import (
	"net/http"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	users *services.UserService
}

func NewUserController(users *services.UserService) *UserController {
	return &UserController{users: users}
}

func (c *UserController) Create(ctx *gin.Context) {
	var in services.UserInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	user, err := c.users.Create(companyFromContext(ctx), userFromContext(ctx), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, user)
}

func (c *UserController) List(ctx *gin.Context) {
	users, err := c.users.List(companyFromContext(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, users)
}

func (c *UserController) Get(ctx *gin.Context) {
	user, err := c.users.Get(companyFromContext(ctx), ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "usuário não encontrado"})
		return
	}
	ctx.JSON(http.StatusOK, user)
}

func (c *UserController) Update(ctx *gin.Context) {
	var in services.UserInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	user, err := c.users.Update(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id"), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, user)
}

func (c *UserController) Delete(ctx *gin.Context) {
	if err := c.users.Delete(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id")); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}
