package app

import (
	"fmt"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/controller"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/route"
	"go-starter-template/internal/service"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	db         *gorm.DB
	web        *fiber.App
	log        *logrus.Logger
	config     *env.Config
	validation *validation.Validation
	redis      *redis.Client
}

func NewApp(log *logrus.Logger, config *env.Config, db *gorm.DB, web *fiber.App, validation *validation.Validation, redis *redis.Client) *BootstrapConfig {
	return &BootstrapConfig{db, web, log, config, validation, redis}
}

func (app *BootstrapConfig) Bootstrap() {
	// setup repositories
	userRepository := repository.NewUserRepository(app.db)
	blacklistRepository := repository.NewRedisTokenBlacklist(app.redis)

	// setup use service
	jwtService := service.NewJwtService(app.log, app.config)
	blacklistService := service.NewBlacklistService(app.log, jwtService, blacklistRepository)
	authService := service.NewAuthService(app.db, jwtService, userRepository, blacklistService, app.log)
	redisService := service.NewRedisService(app.redis, app.log)
	userService := service.NewUserService(app.db, userRepository, redisService, app.log)

	// setup controller
	welcomeController := controller.NewWelcomeController()
	authController := controller.NewAuthController(authService, app.log, app.validation, app.config)
	userController := controller.NewUserController(userService, app.log)

	// setup middleware
	authMiddleware := middleware.AuthMiddleware(jwtService, blacklistService, app.log)

	// setup route
	routeConfig := route.NewRouteConfig(app.web)
	routeConfig.WelcomeRoutes(welcomeController)
	routeConfig.RegisterAuthRoutes(authController)
	routeConfig.RegisterUserRoutes(userController, authMiddleware)
}

func (app *BootstrapConfig) Run() {
	app.Bootstrap()
	err := app.web.Listen(fmt.Sprintf(":%d", app.config.Web.Port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
