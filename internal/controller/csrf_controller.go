package controller

import (
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/utils/csrfutil"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type CsrfController struct {
	Logger *logrus.Logger
	Tracer trace.Tracer
	Config *env.Config
}

func NewCsrfController(logger *logrus.Logger, config *env.Config) *CsrfController {
	return &CsrfController{
		Logger: logger,
		Tracer: otel.Tracer("CsrfController"),
		Config: config,
	}
}

func (c *CsrfController) GenerateCsrfToken(ctx *fiber.Ctx) error {
	_, span := c.Tracer.Start(ctx.UserContext(), "GenerateCsrfToken")
	defer span.End()

	var req dto.CsrfRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Logger.WithError(err).Error("Failed to parse csrf token request")
		return fiber.ErrBadRequest
	}

	csrfToken, err := csrfutil.GenerateCsrfToken(req.Path, c.Config.JWT.CsrfSecret)
	if err != nil {
		return err
	}

	return ctx.JSON(dto.WebResponse[dto.CsrfResponse]{
		Data: dto.CsrfResponse{
			CsrfToken: csrfToken,
		},
	})
}
