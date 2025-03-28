package monitoring

import (
	"context"
	"go-starter-template/internal/config/env"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Monitoring struct {
	Tracer *trace.TracerProvider
}

func NewMonitoring(log *logrus.Logger, config *env.Config) *Monitoring {
	headers := map[string]string{
		"content-type": "application/json",
	}

	// Create an OTLP exporter
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(config.Monitoring.Otel.Host),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		log.WithError(err).Fatal("Failed to create OTLP exporter")
	}

	// Create a new tracer provider with the exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.App.Name),
		)),
	)

	// Set the global tracer provider
	otel.SetTracerProvider(tp)

	return &Monitoring{
		Tracer: tp,
	}
}

func (m *Monitoring) Shutdown() error {
	return m.Tracer.Shutdown(context.Background())
}
