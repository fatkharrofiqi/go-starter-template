package controller

import (
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type CsrfController struct {
	Logger      *logrus.Logger
	Tracer      trace.Tracer
	CsrfService *service.CsrfService
}

func NewCsrfController(logger *logrus.Logger, config *env.Config, csrfService *service.CsrfService) *CsrfController {
	return &CsrfController{
		Logger:      logger,
		Tracer:      otel.Tracer("CsrfController"),
		CsrfService: csrfService,
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

	csrfToken, err := c.CsrfService.GenerateCsrfToken(req.Path)
	if err != nil {
		return err
	}

	return ctx.JSON(dto.WebResponse[dto.CsrfResponse]{
		Data: dto.CsrfResponse{
			CsrfToken: csrfToken,
		},
	})
}
