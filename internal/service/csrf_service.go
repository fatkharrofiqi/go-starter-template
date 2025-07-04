package service

import (
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/utils/csrfutil"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

type CsrfService struct {
	Config *env.Config
	Log    *logrus.Logger
	Tracer *trace.Tracer
}

func NewCsrfService(config *env.Config, log *logrus.Logger, tracer *trace.Tracer) *CsrfService {
	return &CsrfService{
		Config: config,
		Log:    log,
		Tracer: tracer,
	}
}

func (csrfService *CsrfService) GenerateCsrfToken(path string) (*string, error) {
	claims, err := csrfutil.GenerateCsrfToken(path, csrfService.Config.JWT.Secret)
	if err != nil {
		return nil, err
	}

	return &claims, err
}
