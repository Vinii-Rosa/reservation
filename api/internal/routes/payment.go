package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerPayment(admin *gin.RouterGroup, h Handlers) {
	admin.GET("/payment-histories", middleware.RequireRole(models.RoleAdmin, models.RoleDev, models.RoleCashier), h.Payment.ListHistories)
	admin.POST("/tables/:id/pay", middleware.RequireRole(models.RoleAdmin, models.RoleDev, models.RoleCashier), h.Payment.PayTable)
}
