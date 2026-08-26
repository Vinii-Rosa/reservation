package controllers

import (
	"net/http"
	"strconv"
	"time"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type PublicReservationController struct {
	reservations *services.ReservationService
}

func NewPublicReservationController(reservations *services.ReservationService) *PublicReservationController {
	return &PublicReservationController{reservations: reservations}
}

func (c *PublicReservationController) CompanySchedule(ctx *gin.Context) {
	schedule, err := c.reservations.PublicSchedule(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, schedule)
}

func (c *PublicReservationController) Availability(ctx *gin.Context) {
	dateStr := ctx.Query("date")
	partySizeStr := ctx.Query("party_size")
	if partySizeStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "party_size é obrigatório"})
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "date inválida, use YYYY-MM-DD"})
		return
	}
	partySize, err := strconv.Atoi(partySizeStr)
	if err != nil || partySize < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "party_size inválido"})
		return
	}
	slots, err := c.reservations.Availability(ctx.Param("id"), date, partySize)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, slots)
}

func (c *PublicReservationController) CreateReservation(ctx *gin.Context) {
	var in services.CreateReservationInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	res, err := c.reservations.CreatePublic(ctx.Param("id"), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, res)
}

func (c *PublicReservationController) GetReservation(ctx *gin.Context) {
	res, err := c.reservations.GetByPublicToken(ctx.Param("public_token"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "reserva não encontrada"})
		return
	}
	ctx.JSON(http.StatusOK, res)
}
