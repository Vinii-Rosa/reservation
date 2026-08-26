package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerMenuItem(admin *gin.RouterGroup, h Handlers) {
	menu := admin.Group("/menu-items")
	menu.Use(middleware.RequireRole(models.RoleAdmin, models.RoleDev))
	menu.POST("", h.MenuItem.Create)
	menu.GET("", h.MenuItem.List)
	menu.GET("/:id", h.MenuItem.Get)
	menu.PUT("/:id", h.MenuItem.Update)
	menu.DELETE("/:id", h.MenuItem.Delete)
}
