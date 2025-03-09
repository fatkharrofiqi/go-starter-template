package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/logutil"
	"math"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	Log         *logrus.Logger
	UserService *service.UserService
}

func NewUserController(logger *logrus.Logger, userService *service.UserService) *UserController {
	return &UserController{logger, userService}
}

func (c *UserController) Me(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Log, ctx, "Me").Info("Processing me request")

	auth := middleware.GetUser(ctx)

	user, err := c.UserService.GetUser(ctx.UserContext(), auth.UUID)
	if err != nil {
		c.Log.WithError(err).Error("user not found")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.UserResponse]{
		Data: user,
	})
}

func (c *UserController) List(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Log, ctx, "List").Info("Processing list request")

	req := dto.SearchUserRequest{
		Name:  ctx.Query("name"),
		Email: ctx.Query("email"),
		Page:  ctx.QueryInt("page"),
		Size:  ctx.QueryInt("size"),
	}

	users, total, err := c.UserService.Search(ctx.UserContext(), &req)
	if err != nil {
		c.Log.WithError(err).Error("error searching user")
		return err
	}

	return ctx.JSON(dto.WebResponse[[]dto.UserResponse]{
		Data: users,
		Paging: &dto.PageMetadata{
			Page:      req.Page,
			Size:      req.Size,
			TotalItem: total,
			TotalPage: int64(math.Ceil(float64(total) / float64(req.Size))),
		},
	})
}
