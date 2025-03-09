package route

import (
	"go-starter-template/internal/controller"

	"github.com/gofiber/fiber/v2"
)

// RouteConfig handles route registration
type RouteConfig struct {
	App *fiber.App
}

// NewRouteConfig initializes the router
func NewRouteConfig(app *fiber.App) *RouteConfig {
	return &RouteConfig{app}
}

func (r *RouteConfig) WelcomeRoutes(welcomeController *controller.WelcomeController) {
	r.App.Get("/", welcomeController.Hello)
}

// RegisterAuthRoutes defines authentication routes
func (r *RouteConfig) RegisterAuthRoutes(authController *controller.AuthController) {
	auth := r.App.Group("/api/auth")
	{
		auth.Post("/register", authController.Register)
		auth.Post("/login", authController.Login)
		auth.Post("/refresh-token", authController.RefreshToken)
	}
}

// RegisterUserRoutes defines user-related routes with authentication middleware
func (r *RouteConfig) RegisterUserRoutes(userController *controller.UserController, authMiddleware fiber.Handler) {
	user := r.App.Group("/api/users")
	user.Use(authMiddleware)
	{
		user.Get("/", userController.List)
		user.Get("/me", userController.Me)
	}
}
