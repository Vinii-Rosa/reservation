package controllers

import (
	"net/http"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type TableOrderController struct {
	tableOrders *services.TableOrderService
}

func NewTableOrderController(tableOrders *services.TableOrderService) *TableOrderController {
	return &TableOrderController{tableOrders: tableOrders}
}

func (c *TableOrderController) ListPendingTableOrders(ctx *gin.Context) {
	list, err := c.tableOrders.ListPendingTableOrders(companyFromContext(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, list)
}

func (c *TableOrderController) GetTableOrderSummary(ctx *gin.Context) {
	data, err := c.tableOrders.GetTableOrderSummary(companyFromContext(ctx), ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, data)
}
