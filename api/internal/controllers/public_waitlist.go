package controllers

import (
	"net/http"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type PublicWaitlistController struct {
	waitlist *services.WaitlistService
}

func NewPublicWaitlistController(waitlist *services.WaitlistService) *PublicWaitlistController {
	return &PublicWaitlistController{waitlist: waitlist}
}

func (c *PublicWaitlistController) Join(ctx *gin.Context) {
	var in services.JoinWaitlistInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	status, err := c.waitlist.Join(ctx.Param("token"), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, status)
}

func (c *PublicWaitlistController) Status(ctx *gin.Context) {
	status, err := c.waitlist.GetStatus(ctx.Param("token"), ctx.Param("entry_id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, status)
}
