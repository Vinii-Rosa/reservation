package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerSystemEvent(admin *gin.RouterGroup, h Handlers) {
	admin.GET("/system-events", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.SystemEvent.List)
}
