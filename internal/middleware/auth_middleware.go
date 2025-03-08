package middleware

import (
	"go-starter-template/internal/utils/jwtutil"
	"go-starter-template/internal/utils/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func NewAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.ErrorResponse(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		tokenString := strings.Split(authHeader, "Bearer ")[1]
		claims, err := jwtutil.ValidateToken(tokenString, secret)
		if err != nil {
			response.ErrorResponse(c, http.StatusUnauthorized, "Invalid token", err)
			c.Abort()
			return
		}

		c.Set("uid", claims.UID)
		c.Next()
	}
}
