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
	logger      *logrus.Logger
	csrfService *service.CsrfService
	tracer      trace.Tracer
}

func NewCsrfController(logger *logrus.Logger, config *env.Config, csrfService *service.CsrfService) *CsrfController {
	return &CsrfController{logger, csrfService, otel.Tracer("CsrfController")}
}

func (c *CsrfController) GenerateCsrfToken(ctx *fiber.Ctx) error {
	_, span := c.tracer.Start(ctx.UserContext(), "GenerateCsrfToken")
	defer span.End()

	var req dto.CsrfRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.logger.WithError(err).Error("Failed to parse csrf token request")
		return fiber.ErrBadRequest
	}

	csrfToken, err := c.csrfService.GenerateCsrfToken(req.Path)
	if err != nil {
		return err
	}

	return ctx.JSON(dto.WebResponse[dto.CsrfResponse]{
		Data: dto.CsrfResponse{
			CsrfToken: csrfToken,
		},
	})
}
