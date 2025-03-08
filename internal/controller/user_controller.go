package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/logutil"
	"go-starter-template/internal/utils/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	Log         *logrus.Logger
	UserService *service.UserService
}

func NewUserController(logger *logrus.Logger, userService *service.UserService) *UserController {
	return &UserController{logger, userService}
}

func (c *UserController) Me(ctx *gin.Context) {
	logutil.RequestEntry(c.Log, ctx, "Me").Info("Processing me request")
	uid, _ := ctx.Get("uid")
	user, err := c.UserService.GetUser(ctx, uid.(string))
	if err != nil {
		response.ErrorResponse(ctx, http.StatusNotFound, "User not found", err)
	}
	response.SuccessResponse(ctx, http.StatusOK, dto.UserResponse{
		UID:       user.UID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	})
}
