package controller

import (
	"go-starter-template/internal/utils/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type WelcomeController struct{}

// NewWelcomeController creates a new instance of WelcomeController
func NewWelcomeController() *WelcomeController {
	return &WelcomeController{}
}

func (r *WelcomeController) Hello(ctx *gin.Context) {
	response.SuccessResponse(ctx, http.StatusOK, gin.H{
		"message": "Welcome to Go Starter API!",
	})
}
