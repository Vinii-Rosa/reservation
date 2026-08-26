package controllers

import (
	"net/http"
	"time"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type SystemEventController struct {
	events *services.SystemEventService
}

func NewSystemEventController(events *services.SystemEventService) *SystemEventController {
	return &SystemEventController{events: events}
}

func (c *SystemEventController) List(ctx *gin.Context) {
	var from, to *time.Time
	if v := ctx.Query("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			from = &t
		}
	}
	if v := ctx.Query("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			to = &t
		}
	}
	events, err := c.events.List(companyFromContext(ctx), ctx.Query("type"), ctx.Query("actor_type"), from, to)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, events)
}
