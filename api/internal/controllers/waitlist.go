package controllers

import (
	"net/http"

	"reservation/api/internal/models"
	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type WaitlistController struct {
	waitlist *services.WaitlistService
}

func NewWaitlistController(waitlist *services.WaitlistService) *WaitlistController {
	return &WaitlistController{waitlist: waitlist}
}

func (c *WaitlistController) List(ctx *gin.Context) {
	entries, err := c.waitlist.List(companyFromContext(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, entries)
}

type waitlistPatch struct {
	Status models.WaitlistStatus `json:"status"`
}

func (c *WaitlistController) Update(ctx *gin.Context) {
	var in waitlistPatch
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	entry, err := c.waitlist.UpdateStatus(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id"), in.Status)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, entry)
}
