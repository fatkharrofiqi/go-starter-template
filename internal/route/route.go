package route

import (
	"go-starter-template/internal/controller"

	"github.com/gin-gonic/gin"
)

// RouteConfig handles route registration
type RouteConfig struct {
	App *gin.Engine
}

// NewRouteConfig initializes the router
func NewRouteConfig(app *gin.Engine) *RouteConfig {
	return &RouteConfig{App: app}
}

// RegisterAuthRoutes defines authentication routes
func (r *RouteConfig) RegisterAuthRoutes(authController *controller.AuthController) {
	auth := r.App.Group("/api/auth")
	{
		auth.POST("/register", authController.Register)
		auth.POST("/login", authController.Login)
		auth.POST("/refresh-token", authController.RefreshToken)
	}
}

// RegisterUserRoutes defines user-related routes with authentication middleware
func (r *RouteConfig) RegisterUserRoutes(userController *controller.UserController, authMiddleware gin.HandlerFunc) {
	user := r.App.Group("/api/users")
	user.Use(authMiddleware)
	{
		user.GET("/me", userController.Me)
	}
}
