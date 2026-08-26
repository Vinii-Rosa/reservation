package controllers

import (
	"net/http"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type PublicTableController struct {
	tables      *services.TableService
	tableOrders *services.TableOrderService
}

func NewPublicTableController(tables *services.TableService, tableOrders *services.TableOrderService) *PublicTableController {
	return &PublicTableController{tables: tables, tableOrders: tableOrders}
}

func (c *PublicTableController) GetTable(ctx *gin.Context) {
	data, err := c.tables.GetPublicWithMenu(ctx.Param("token"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "mesa não encontrada"})
		return
	}
	ctx.JSON(http.StatusOK, data)
}

type publicTableOrderBody struct {
	GuestName string                         `json:"guest_name"`
	Items     []services.TableOrderItemInput `json:"items"`
}

func (c *PublicTableController) CreateTableOrder(ctx *gin.Context) {
	var in publicTableOrderBody
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	tableOrder, err := c.tableOrders.CreatePublic(ctx.Param("token"), in.GuestName, services.CreateTableOrderInput{Items: in.Items})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, tableOrder)
}
