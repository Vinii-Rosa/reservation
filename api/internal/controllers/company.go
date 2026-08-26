package controllers

import (
	"net/http"

	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
)

type CompanyController struct {
	companies *services.CompanyService
}

func NewCompanyController(companies *services.CompanyService) *CompanyController {
	return &CompanyController{companies: companies}
}

func (c *CompanyController) Create(ctx *gin.Context) {
	var in services.CreateCompanyInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	company, err := c.companies.Create(userFromContext(ctx), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, company)
}

func (c *CompanyController) Get(ctx *gin.Context) {
	company, err := c.companies.Get(companyFromContext(ctx))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "companhia não encontrada"})
		return
	}
	ctx.JSON(http.StatusOK, company)
}

func (c *CompanyController) Update(ctx *gin.Context) {
	var in services.UpdateCompanyInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	company, err := c.companies.Update(companyFromContext(ctx), userFromContext(ctx), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, company)
}

func (c *CompanyController) ListConfigs(ctx *gin.Context) {
	list, err := c.companies.ListConfigs(companyFromContext(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, list)
}

func (c *CompanyController) UpdateConfig(ctx *gin.Context) {
	var in services.UpdateCompanyConfigInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	list, err := c.companies.UpdateConfig(companyFromContext(ctx), ctx.Param("key"), in.Value)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, list)
}

func (c *CompanyController) UpdateSchedule(ctx *gin.Context) {
	var in services.UpdateScheduleInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}
	company, err := c.companies.UpdateSchedule(companyFromContext(ctx), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, company)
}
