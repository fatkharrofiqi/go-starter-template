package config

import (
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/monitoring"
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/controller"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/route"
	"go-starter-template/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	DB         *gorm.DB
	App        *fiber.App
	Log        *logrus.Logger
	Config     *env.Config
	Validation *validation.Validation
	Monitoring *monitoring.Monitoring
	Redis      *redis.Client
}

func Bootstrap(app *BootstrapConfig) {
	// setup repositories
	userRepository := repository.NewUserRepository()
	blacklistRepository := repository.NewRedisTokenBlacklist(app.Redis)

	// setup use service
	jwtService := service.NewJwtService(app.Config, blacklistRepository)
	blacklistService := service.NewBlacklistService(blacklistRepository, jwtService)
	authService := service.NewAuthService(app.DB, jwtService, userRepository, blacklistService, app.Log)
	redisService := service.NewRedisService(app.Redis, app.Log)
	userService := service.NewUserService(app.DB, userRepository, redisService, app.Log)

	// setup controller
	welcomeController := controller.NewWelcomeController()
	authController := controller.NewAuthController(authService, app.Log, app.Validation)
	userController := controller.NewUserController(userService, app.Log)

	// setup middleware
	authMiddleware := middleware.AuthMiddleware(jwtService, blacklistService, app.Log)

	// setup route
	routeConfig := route.NewRouteConfig(app.App)
	routeConfig.WelcomeRoutes(welcomeController)
	routeConfig.RegisterAuthRoutes(authController)
	routeConfig.RegisterUserRoutes(userController, authMiddleware)
}
