package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/apperrors"
	"go-starter-template/internal/utils/logutil"
	"math"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	UserService *service.UserService
	Logger      *logrus.Logger
}

func NewUserController(userService *service.UserService, logger *logrus.Logger) *UserController {
	return &UserController{userService, logger}
}

func (c *UserController) Me(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Logger, ctx, "Me").Info("Processing me request")

	auth := middleware.GetUser(ctx)

	user, err := c.UserService.GetUser(ctx.UserContext(), auth.UUID)
	if err != nil {
		c.Logger.WithError(err).Error("user not found")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.UserResponse]{
		Data: user,
	})
}

func (c *UserController) List(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Logger, ctx, "List").Info("Processing list user request")

	req := new(dto.SearchUserRequest)
	if err := ctx.QueryParser(req); err != nil {
		c.Logger.WithError(err).Error("failed to parse request query")
		return apperrors.ErrBadRequest
	}
	req.SetDefault()

	users, total, err := c.UserService.Search(ctx.UserContext(), req)
	if err != nil {
		c.Logger.WithError(err).Error("error searching user")
		return err
	}

	return ctx.JSON(dto.WebResponse[[]*dto.UserResponse]{
		Data: users,
		Paging: &dto.PageMetadata{
			Page:      req.Page,
			Size:      req.Size,
			TotalItem: total,
			TotalPage: int64(math.Ceil(float64(total) / float64(req.Size))),
		},
	})
}
