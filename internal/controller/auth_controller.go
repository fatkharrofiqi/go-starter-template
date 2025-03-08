package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/errwrap"
	"go-starter-template/internal/utils/logutil"
	"go-starter-template/internal/utils/response"
	"go-starter-template/internal/utils/validatorutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

type AuthController struct {
	AuthService *service.AuthService
	Log         *logrus.Logger
	Validator   *validator.Validate
}

func NewAuthController(authService *service.AuthService, logger *logrus.Logger, validator *validator.Validate) *AuthController {
	return &AuthController{authService, logger, validator}
}

func (c *AuthController) Login(ctx *gin.Context) {
	// Log entry of the method with request details
	logutil.RequestEntry(c.Log, ctx, "Login").Info("Processing login request")

	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logutil.Error(c.Log, "Invalid request", err, logrus.Fields{
			"email": req.Email,
		})
		response.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request")
		return
	}

	if errs := validatorutil.ValidateStruct(c.Validator, req); len(errs) > 0 {
		logutil.Error(c.Log, "Validation failed", nil, logrus.Fields{"email": req.Email, "errors": errs})
		response.ErrorResponse(ctx, http.StatusBadRequest, "Validation failed", errs)
		return
	}

	token, err := c.AuthService.Login(ctx, req)
	if err != nil {
		logutil.Error(c.Log, "Invalid credentials", err, logrus.Fields{
			"email": req.Email,
		})
		response.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	response.SuccessResponse(ctx, http.StatusOK, dto.LoginResponse{
		TokenResponse: *token,
	})
}

func (c *AuthController) Register(ctx *gin.Context) {
	// Log entry of the method with request details
	logutil.RequestEntry(c.Log, ctx, "Register").Info("Processing registration request")

	var req dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logutil.Error(c.Log, "Invalid request", err)
		response.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request")
		return
	}

	if errs := validatorutil.ValidateStruct(c.Validator, req); len(errs) > 0 {
		logutil.Error(c.Log, "Validation failed", nil, logrus.Fields{"email": req.Email, "errors": errs})
		response.ErrorResponse(ctx, http.StatusBadRequest, "Validation failed", errs)
		return
	}

	if err := c.AuthService.Register(ctx, req); err != nil {
		logutil.Error(c.Log, "Registration failed", err)
		switch {
		case errwrap.IsErrorType(err, errwrap.ErrDataExists):
			response.ErrorResponse(ctx, http.StatusConflict, "Registration failed", err)
		default:
			response.ErrorResponse(ctx, http.StatusInternalServerError, "Registration failed", err)
		}
		return
	}

	response.SuccessResponse(ctx, http.StatusCreated, nil)
}

func (c *AuthController) RefreshToken(ctx *gin.Context) {
	logutil.RequestEntry(c.Log, ctx, "RefreshToken").Info("Processing refresh token request")

	var req dto.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logutil.Error(c.Log, "Invalid request", err)
		response.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request")
		return
	}

	if errs := validatorutil.ValidateStruct(c.Validator, req); len(errs) > 0 {
		logutil.Error(c.Log, "Validation failed", nil, logrus.Fields{"errors": errs})
		response.ErrorResponse(ctx, http.StatusBadRequest, "Validation failed", errs)
		return
	}

	token, err := c.AuthService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		logutil.Error(c.Log, "Invalid credentials", err)
		response.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	response.SuccessResponse(ctx, http.StatusOK, dto.RefreshTokenResponse{
		TokenResponse: *token,
	})
}
