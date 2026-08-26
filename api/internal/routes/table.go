package routes

import (
	"reservation/api/internal/middleware"
	"reservation/api/internal/models"

	"github.com/gin-gonic/gin"
)

func registerTable(admin *gin.RouterGroup, public *gin.RouterGroup, h Handlers) {
	tables := admin.Group("/tables")
	tables.Use(middleware.RequireRole(models.RoleAdmin, models.RoleDev, models.RoleCashier))
	tables.POST("", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Table.Create)
	tables.GET("", h.Table.List)
	tables.GET("/:id", h.Table.Get)
	tables.PUT("/:id", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Table.Update)
	tables.DELETE("/:id", middleware.RequireRole(models.RoleAdmin, models.RoleDev), h.Table.Delete)
	tables.PATCH("/:id/status", h.Table.SetStatus)
	tables.GET("/:id/qr", h.Table.QR)
	tables.GET("/:id/order-summary", h.TableOrder.GetTableOrderSummary)

	public.GET("/tables/:token", h.PublicTable.GetTable)
	public.POST("/tables/:token/table-orders", h.PublicTable.CreateTableOrder)
}
