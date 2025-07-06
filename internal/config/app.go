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
	tokenBlacklistRepository := repository.NewRedisTokenBlacklist(app.Redis)

	// setup use service
	authService := service.NewAuthService(app.DB, userRepository, tokenBlacklistRepository, app.Config, app.Log)
	redisService := service.NewRedisService(app.Redis, app.Log)
	userService := service.NewUserService(app.DB, userRepository, redisService, app.Log)

	// setup controller
	welcomeController := controller.NewWelcomeController()
	authController := controller.NewAuthController(authService, app.Log, app.Validation)
	userController := controller.NewUserController(userService, app.Log)
	csrfController := controller.NewCsrfController(app.Log, app.Config)

	// setup middleware
	authMiddleware := middleware.AuthMiddleware(app.Config.JWT.Secret, app.Redis, app.Log, tokenBlacklistRepository)
	csrfMiddleware := middleware.CsrfMiddleware(app.Config.JWT.CsrfSecret, app.Log, tokenBlacklistRepository)

	// setup route
	routeConfig := route.NewRouteConfig(app.App)
	routeConfig.WelcomeRoutes(welcomeController)
	routeConfig.RegisterAuthRoutes(authController)
	routeConfig.RegisterCsrfRoute(csrfController, authMiddleware)
	routeConfig.RegisterUserRoutes(userController, authMiddleware, csrfMiddleware)
}
