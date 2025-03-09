package config

import (
	"go-starter-template/internal/controller"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/route"
	"go-starter-template/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	DB        *gorm.DB
	App       *fiber.App
	Log       *logrus.Logger
	Viper     *viper.Viper
	Validator *validator.Validate
}

func Bootstrap(config *BootstrapConfig) {
	// setup repositories
	userRepository := repository.NewUserRepository()

	// setup use service
	authService := service.NewAuthService(config.DB, userRepository, config.Viper, config.Log)
	userService := service.NewUserService(config.DB, userRepository, config.Log)

	// setup controller
	welcomeController := controller.NewWelcomeController()
	authController := controller.NewAuthController(authService, config.Log, config.Validator)
	userController := controller.NewUserController(config.Log, userService)

	// setup middleware
	authMiddleware := middleware.AuthMiddleware(config.Viper.GetString("jwt.secret"), config.Log)

	// setup route
	routeConfig := route.NewRouteConfig(config.App)
	routeConfig.WelcomeRoutes(welcomeController)
	routeConfig.RegisterAuthRoutes(authController)
	routeConfig.RegisterUserRoutes(userController, authMiddleware)
}
