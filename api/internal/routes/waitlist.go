package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerWaitlist(admin *gin.RouterGroup, public *gin.RouterGroup, h Handlers) {
	waitlist := admin.Group("/waitlist")
	waitlist.Use(middleware.RequireRole(models.RoleAdmin, models.RoleDev))
	waitlist.GET("", h.Waitlist.List)
	waitlist.PATCH("/:id", h.Waitlist.Update)

	public.POST("/waitlist/:token", h.PublicWaitlist.Join)
	public.GET("/waitlist/:token/:entry_id", h.PublicWaitlist.Status)
}
