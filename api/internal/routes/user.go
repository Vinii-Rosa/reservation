package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerUser(admin *gin.RouterGroup, h Handlers) {
	users := admin.Group("/users")
	users.Use(middleware.RequireRole(models.RoleAdmin, models.RoleDev))
	users.POST("", h.User.Create)
	users.GET("", h.User.List)
	users.GET("/:id", h.User.Get)
	users.PUT("/:id", h.User.Update)
	users.DELETE("/:id", h.User.Delete)
}
