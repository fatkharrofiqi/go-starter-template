package config

import (
	"go-starter-template/internal/controller"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/route"
	"go-starter-template/internal/service"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	DB        *gorm.DB
	App       *gin.Engine
	Log       *logrus.Logger
	Viper     *viper.Viper
	Producer  *kafka.Producer
	Validator *validator.Validate
}

func Bootstrap(config *BootstrapConfig) {
	// setup repositories
	userRepository := repository.NewUserRepository()

	// setup use service
	authService := service.NewAuthService(config.DB, userRepository, config.Viper)
	userService := service.NewUserService(config.DB, userRepository)

	// setup controller
	authController := controller.NewAuthController(authService, config.Log, config.Validator)
	userController := controller.NewUserController(config.Log, userService)

	// setup middleware
	authMiddleware := middleware.NewAuthMiddleware(config.Viper.GetString("jwt.secret"))

	// setup route
	routeConfig := route.NewRouteConfig(config.App)
	routeConfig.RegisterAuthRoutes(authController)
	routeConfig.RegisterUserRoutes(userController, authMiddleware)
}
