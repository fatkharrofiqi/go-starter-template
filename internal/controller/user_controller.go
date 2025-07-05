package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/apperrors"
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type UserController struct {
	UserService  *service.UserService
	RedisService *service.RedisService
	Logger       *logrus.Logger
	Tracer       trace.Tracer
}

func NewUserController(userService *service.UserService, redisService *service.RedisService, logger *logrus.Logger) *UserController {
	return &UserController{userService, redisService, logger, otel.Tracer("UserController")}
}

func (c *UserController) Me(ctx *fiber.Ctx) error {
	userContext, span := c.Tracer.Start(ctx.UserContext(), "Me")
	defer span.End()

	auth := middleware.GetUser(ctx)
	cacheKey := "user:me:" + auth.UUID

	var response dto.WebResponse[*dto.UserResponse]

	// 🔍 Check Redis cache first
	if cached, found := c.RedisService.Get(userContext, cacheKey); found {
		c.Logger.Info("user profile retrieved from Redis cache")
		return ctx.Type("json").SendString(cached)
	}

	// 🗃️ Cache miss: fetch from DB
	user, err := c.UserService.GetUser(userContext, auth.UUID)
	if err != nil {
		c.Logger.WithError(err).Error("user not found")
		return err
	}

	response = dto.WebResponse[*dto.UserResponse]{
		Data: user,
	}

	// 💾 Store in Redis with TTL
	c.RedisService.Set(userContext, cacheKey, response, 10*time.Minute)

	return ctx.JSON(response)
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
