package apperrors

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

var (
	// Authentication Errors
	ErrInvalidEmailOrPassword = errors.New("invalid email or password")
	ErrInvalidToken           = errors.New("invalid token")
	ErrTokenInvalidation      = errors.New("failed to invalidate token")
	ErrTokenBlacklisted       = errors.New("failed token is blacklist")
	ErrAuthorizationHeader    = errors.New("authorization header is required")
	ErrTokenIsExpired         = errors.New("token is expired")

	// Access Urls Errors
	ErrCsrfTokenHeader      = errors.New("csrf token is required")
	ErrCsrfTokenInvalidPath = errors.New("csrf token is invalid for this url")
	ErrCsrfTokenIsExpired   = errors.New("csrf token is expired")

	// User Errors
	ErrUserNotFound     = errors.New("user not found")
	ErrUserSearchFailed = errors.New("failed to retrieve users")

	// Registration Errors
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrPasswordEncryption  = errors.New("password encryption error")
	ErrUserCreationFailed  = errors.New("user creation failed")
	ErrDatabaseTransaction = errors.New("database transaction failed")
	ErrDatabaseError       = errors.New("database error")

	// Token Errors
	ErrAccessTokenGeneration  = errors.New("could not generate access token")
	ErrRefreshTokenGeneration = errors.New("could not generate refresh token")

	// Common Errors
	ErrBadRequest = errors.New("bad request")
)

// errorStatusMap maps application errors to their respective HTTP status codes
var errorStatusMap = map[error]int{
	// 401 Unauthorized Errors
	ErrInvalidEmailOrPassword: fiber.StatusUnauthorized,
	ErrInvalidToken:           fiber.StatusUnauthorized,
	ErrTokenInvalidation:      fiber.StatusUnauthorized,
	ErrTokenBlacklisted:       fiber.StatusUnauthorized,
	ErrAuthorizationHeader:    fiber.StatusUnauthorized,
	ErrCsrfTokenHeader:        fiber.StatusUnauthorized,
	ErrCsrfTokenInvalidPath:   fiber.StatusUnauthorized,
	ErrCsrfTokenIsExpired:     fiber.StatusUnauthorized,
	ErrTokenIsExpired:         fiber.StatusUnauthorized,

	// 409 Conflict Errors
	ErrUserAlreadyExists: fiber.StatusConflict,

	// 500 Internal Server Errors
	ErrDatabaseError:          fiber.StatusInternalServerError,
	ErrDatabaseTransaction:    fiber.StatusInternalServerError,
	ErrPasswordEncryption:     fiber.StatusInternalServerError,
	ErrUserCreationFailed:     fiber.StatusInternalServerError,
	ErrAccessTokenGeneration:  fiber.StatusInternalServerError,
	ErrRefreshTokenGeneration: fiber.StatusInternalServerError,

	// 404 Not Found Errors
	ErrUserNotFound:     fiber.StatusNotFound,
	ErrUserSearchFailed: fiber.StatusNotFound,
	ErrBadRequest:       fiber.StatusBadRequest,
}

// GetHTTPStatus retrieves the HTTP status code for a given error.
func GetHTTPStatus(err error) (int, bool) {
	statusCode, exists := errorStatusMap[err]
	return statusCode, exists
}
