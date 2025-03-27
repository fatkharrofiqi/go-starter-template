package config

import (
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/controller"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/route"
	"go-starter-template/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	DB         *gorm.DB
	App        *fiber.App
	Log        *logrus.Logger
	Config     *env.Config
	Validation *validation.Validation
}

func Bootstrap(app *BootstrapConfig) {
	// setup repositories
	userRepository := repository.NewUserRepository()
	tokenBlacklistRepository := repository.NewTokenBlacklist()

	// setup use service
	authService := service.NewAuthService(app.DB, userRepository, tokenBlacklistRepository, app.Config, app.Log)
	userService := service.NewUserService(app.DB, userRepository, app.Log)

	// setup controller
	welcomeController := controller.NewWelcomeController()
	authController := controller.NewAuthController(authService, app.Log, app.Validation)
	userController := controller.NewUserController(userService, app.Log)

	// setup middleware
	authMiddleware := middleware.AuthMiddleware(app.Config.JWT.Secret, app.Log, tokenBlacklistRepository)

	// setup route
	routeConfig := route.NewRouteConfig(app.App)
	routeConfig.WelcomeRoutes(welcomeController)
	routeConfig.RegisterAuthRoutes(authController)
	routeConfig.RegisterUserRoutes(userController, authMiddleware)
}
