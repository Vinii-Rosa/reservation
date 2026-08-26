package controllers

import (
	"net/http"

	"reservation/api/internal/models"
	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type TableController struct {
	tables   *services.TableService
	waitlist *services.WaitlistService
}

func NewTableController(tables *services.TableService, waitlist *services.WaitlistService) *TableController {
	return &TableController{tables: tables, waitlist: waitlist}
}

func (c *TableController) Create(ctx *gin.Context) {
	var in services.CreateTableInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	table, err := c.tables.Create(companyFromContext(ctx), userFromContext(ctx), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, table)
}

func (c *TableController) List(ctx *gin.Context) {
	tables, err := c.tables.List(companyFromContext(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, tables)
}

func (c *TableController) Get(ctx *gin.Context) {
	table, err := c.tables.Get(companyFromContext(ctx), ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "mesa não encontrada"})
		return
	}
	ctx.JSON(http.StatusOK, table)
}

func (c *TableController) Update(ctx *gin.Context) {
	var in services.UpdateTableInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	table, err := c.tables.Update(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id"), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, table)
}

func (c *TableController) Delete(ctx *gin.Context) {
	if err := c.tables.Delete(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id")); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}

type setStatusInput struct {
	Status models.TableStatus `json:"status"`
}

func (c *TableController) SetStatus(ctx *gin.Context) {
	var in setStatusInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	table, err := c.tables.SetStatus(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id"), in.Status)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.Status == models.TableStatusAvailable {
		_ = c.waitlist.OnTableFreed(companyFromContext(ctx))
	}
	ctx.JSON(http.StatusOK, table)
}

func (c *TableController) QR(ctx *gin.Context) {
	url, b64, err := c.tables.QRCode(companyFromContext(ctx), ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"url": url, "qr_base64": b64})
}
