package controllers

import (
	"net/http"
	"strconv"
	"time"

	"reservation/api/internal/models"
	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type ReservationController struct {
	reservations *services.ReservationService
}

func NewReservationController(reservations *services.ReservationService) *ReservationController {
	return &ReservationController{reservations: reservations}
}

func (c *ReservationController) Create(ctx *gin.Context) {
	var in services.CreateReservationInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	res, err := c.reservations.CreateAdmin(companyFromContext(ctx), userFromContext(ctx), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, res)
}

func (c *ReservationController) List(ctx *gin.Context) {
	var filter services.ListReservationsFilter
	if v := ctx.Query("date"); v != "" {
		date, err := time.Parse("2006-01-02", v)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "date inválida, use YYYY-MM-DD"})
			return
		}
		filter.Date = &date
	}
	if v := ctx.Query("time"); v != "" {
		clock, err := time.Parse("15:04", v)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "time inválido, use HH:MM"})
			return
		}
		filter.Time = &clock
	}
	if v := ctx.Query("party_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "party_size inválido"})
			return
		}
		filter.PartySize = &n
	}
	if v := ctx.Query("status"); v != "" {
		status := models.ReservationStatus(v)
		if status != models.ReservationStatusPending &&
			status != models.ReservationStatusSeated &&
			status != models.ReservationStatusCompleted &&
			status != models.ReservationStatusCancelled &&
			status != models.ReservationStatusNoShow {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "status inválido"})
			return
		}
		filter.Status = status
	}

	list, err := c.reservations.List(companyFromContext(ctx), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, list)
}

func (c *ReservationController) Get(ctx *gin.Context) {
	res, err := c.reservations.Get(companyFromContext(ctx), ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "reserva não encontrada"})
		return
	}
	ctx.JSON(http.StatusOK, res)
}

func (c *ReservationController) Update(ctx *gin.Context) {
	var in services.UpdateReservationInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	res, err := c.reservations.Update(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id"), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, res)
}

func (c *ReservationController) Delete(ctx *gin.Context) {
	if err := c.reservations.Delete(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id")); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (c *ReservationController) CheckIn(ctx *gin.Context) {
	res, err := c.reservations.CheckIn(companyFromContext(ctx), userFromContext(ctx), ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, res)
}
