package controllers

import (
	"net/http"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type PaymentController struct {
	payments *services.PaymentService
}

func NewPaymentController(payments *services.PaymentService) *PaymentController {
	return &PaymentController{payments: payments}
}

func (c *PaymentController) PayTable(ctx *gin.Context) {
	history, err := c.payments.PayTable(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, history)
}

func (c *PaymentController) ListHistories(ctx *gin.Context) {
	list, err := c.payments.ListHistories(companyFromContext(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, list)
}
