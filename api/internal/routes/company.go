package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerCompany(admin *gin.RouterGroup, h Handlers) {
	admin.POST("/company", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Company.Create)

	owned := admin.Group("/company")
	owned.Use(middleware.RequireCompany())
	owned.GET("", h.Company.Get)
	owned.PATCH("", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Company.Update)
	owned.PATCH("/schedule", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Company.UpdateSchedule)
	owned.GET("/configs", h.Company.ListConfigs)
	owned.PATCH("/configs/:key", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Company.UpdateConfig)
}
