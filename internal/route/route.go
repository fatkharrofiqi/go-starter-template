package route

import (
    "time"
    "go-starter-template/internal/controller"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/limiter"
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
        // Apply rate limiting specifically on login route to mitigate brute-force attempts
        auth.Post("/login",
            limiter.New(limiter.Config{
                Max:        5,
                Expiration: time.Minute,
                KeyGenerator: func(c *fiber.Ctx) string {
                    return c.IP()
                },
            }),
            authController.Login,
        )
        auth.Post("/logout", authController.Logout)
        auth.Post("/refresh-token", authController.RefreshToken)
    }
}

func (r *RouteConfig) RegisterCsrfRoute(csrfController *controller.CsrfController, authMiddleware fiber.Handler) {
	csrf := r.App.Group("/api/csrf")
	{
		csrf.Use(authMiddleware)
		csrf.Post("/", csrfController.GenerateCsrfToken)
	}
}

// RegisterUserRoutes defines user-related routes with authentication middleware
func (r *RouteConfig) RegisterUserRoutes(userController *controller.UserController, authMiddleware fiber.Handler) {
	user := r.App.Group("/api/users")
	{
		user.Use(authMiddleware)
		user.Get("/", userController.List)
		user.Get("/me", userController.Me)
		user.Post("/", userController.Create)
		user.Put("/:uuid", userController.Update)
		user.Delete("/:uuid", userController.Delete)
	}
}
