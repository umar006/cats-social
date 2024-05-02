package handler

import (
	"cats-social/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CatMatchHandler interface {
	CreateCatMatch() gin.HandlerFunc
	GetCatMatches() gin.HandlerFunc
	GetCatMatchByID() gin.HandlerFunc
	UpdateCatMatchByID() gin.HandlerFunc
	DeleteCatMatch() gin.HandlerFunc
}

type catMatchHandler struct {
	catMatchService service.CatMatchService
}

func NewCatMatchHandler(catMatchService service.CatMatchService) CatMatchHandler {
	return &catMatchHandler{
		catMatchService: catMatchService,
	}
}

func (c *catMatchHandler) CreateCatMatch() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := c.catMatchService.CreateCatMatch(ctx, nil)
		if err != nil {
			ctx.JSON(err.Status(), gin.H{
				"message": err.Message(),
			})

			return
		}

		ctx.JSON(http.StatusCreated, gin.H{
			"message": result,
		})
	}
}

func (c *catMatchHandler) GetCatMatches() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Get Cat Matches",
		})
	}
}

func (c *catMatchHandler) GetCatMatchByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Get Cat Match By ID",
		})
	}
}

func (c *catMatchHandler) UpdateCatMatchByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Update Cat Match By ID",
		})
	}
}

func (c *catMatchHandler) DeleteCatMatch() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Delete Cat Match",
		})
	}
}
