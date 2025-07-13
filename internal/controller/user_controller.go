package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/errcode"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type UserController struct {
	userService *service.UserService
	logger      *logrus.Logger
	tracer      trace.Tracer
}

func NewUserController(userService *service.UserService, logger *logrus.Logger) *UserController {
	return &UserController{userService, logger, otel.Tracer("UserController")}
}

func (c *UserController) Me(ctx *fiber.Ctx) error {
	spanCtx, span := c.tracer.Start(ctx.UserContext(), "UserController.Me")
	defer span.End()

	auth := middleware.GetUser(ctx)

	user, err := c.userService.GetUser(spanCtx, auth.UUID)
	if err != nil {
		c.logger.WithContext(spanCtx).WithField("user_id", auth.UUID).WithError(err).Error("failed to fetch user")
		return err
	}

	return ctx.Type("json").SendString(user)
}

func (c *UserController) List(ctx *fiber.Ctx) error {
	spanCtx, span := c.tracer.Start(ctx.UserContext(), "UserController.List")
	defer span.End()

	logger := c.logger.WithContext(spanCtx)

	_, parseSpan := c.tracer.Start(spanCtx, "ParseRequest")
	req := new(dto.SearchUserRequest)
	if err := ctx.QueryParser(req); err != nil {
		parseSpan.End()
		logger.WithError(err).Error("failed to parse request query")
		return errcode.ErrBadRequest
	}
	req.SetDefault()
	parseSpan.End()

	users, totalCount, err := c.userService.Search(spanCtx, req)
	if err != nil {
		logger.WithError(err).Error("error searching user")
		return err
	}

	totalPage := (totalCount + int64(req.Size) - 1) / int64(req.Size)

	return ctx.JSON(dto.WebResponse[[]*dto.UserResponse]{
		Data: users,
		Paging: &dto.PageMetadata{
			Page:      req.Page,
			Size:      req.Size,
			TotalItem: totalCount,
			TotalPage: totalPage,
		}})
}
