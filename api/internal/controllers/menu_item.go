package controllers

import (
	"net/http"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type MenuItemController struct {
	menu *services.MenuItemService
}

func NewMenuItemController(menu *services.MenuItemService) *MenuItemController {
	return &MenuItemController{menu: menu}
}

func (c *MenuItemController) Create(ctx *gin.Context) {
	var in services.CreateMenuItemInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	item, err := c.menu.Create(companyFromContext(ctx), userFromContext(ctx), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

func (c *MenuItemController) List(ctx *gin.Context) {
	items, err := c.menu.List(companyFromContext(ctx), false)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, items)
}

func (c *MenuItemController) Get(ctx *gin.Context) {
	item, err := c.menu.Get(companyFromContext(ctx), ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "item não encontrado"})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *MenuItemController) Update(ctx *gin.Context) {
	var in services.UpdateMenuItemInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	item, err := c.menu.Update(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id"), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *MenuItemController) Delete(ctx *gin.Context) {
	if err := c.menu.Delete(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id")); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}
