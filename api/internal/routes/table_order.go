package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerTableOrder(admin *gin.RouterGroup, h Handlers) {
	admin.GET("/table-orders/pending", middleware.RequireRole(models.RoleAdmin, models.RoleDev, models.RoleCashier), h.TableOrder.ListPendingTableOrders)
}
