package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/apperrors"
	"math"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type UserController struct {
	UserService *service.UserService
	Logger      *logrus.Logger
	Tracer      trace.Tracer
}

func NewUserController(userService *service.UserService, logger *logrus.Logger) *UserController {
	return &UserController{userService, logger, otel.Tracer("UserController")}
}

func (c *UserController) Me(ctx *fiber.Ctx) error {
	userContext, span := c.Tracer.Start(ctx.UserContext(), "Me")
	defer span.End()

	auth := middleware.GetUser(ctx)

	user, err := c.UserService.GetUser(userContext, auth.UUID)
	if err != nil {
		c.Logger.WithError(err).Error("user not found")
		return err
	}

	return ctx.Type("json").SendString(user)
}

func (c *UserController) List(ctx *fiber.Ctx) error {
	userContext, span := c.Tracer.Start(ctx.UserContext(), "List")
	defer span.End()

	req := new(dto.SearchUserRequest)
	if err := ctx.QueryParser(req); err != nil {
		c.Logger.WithError(err).Error("failed to parse request query")
		return apperrors.ErrBadRequest
	}
	req.SetDefault()

	users, total, err := c.UserService.Search(userContext, req)
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
