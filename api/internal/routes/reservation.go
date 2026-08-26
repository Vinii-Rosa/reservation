package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerReservation(admin *gin.RouterGroup, public *gin.RouterGroup, h Handlers) {
	reservations := admin.Group("/reservations")
	reservations.Use(middleware.RequireRole(models.RoleAdmin, models.RoleDev, models.RoleCashier))
	reservations.POST("", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Reservation.Create)
	reservations.GET("", h.Reservation.List)
	reservations.GET("/:id", h.Reservation.Get)
	reservations.PUT("/:id", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Reservation.Update)
	reservations.DELETE("/:id", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Reservation.Delete)
	reservations.PATCH("/:id/check-in", h.Reservation.CheckIn)

	public.GET("/companies/:id/schedule", h.PublicReservation.CompanySchedule)
	public.GET("/companies/:id/availability", h.PublicReservation.Availability)
	public.POST("/companies/:id/reservations", h.PublicReservation.CreateReservation)
	public.GET("/reservations/:public_token", h.PublicReservation.GetReservation)
}
