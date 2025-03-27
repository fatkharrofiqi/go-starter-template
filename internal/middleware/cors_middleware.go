package middleware

import (
	"go-starter-template/internal/config/env"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func Cors(config *env.Config) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     config.Web.Cors.AllowOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		ExposeHeaders:    "Content-Length",
		AllowCredentials: true,
	})
}
