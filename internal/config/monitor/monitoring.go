package monitor

import (
	"context"
	"go-starter-template/internal/config/env"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Monitoring struct {
	tracerProvider *trace.TracerProvider
	loggerProvider *log.LoggerProvider
}

func NewMonitoring(logger *logrus.Logger, config *env.Config) *Monitoring {
	// Create OTLP exporter for traces
	traceExporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(config.Monitoring.Otel.Host),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create OTLP trace exporter")
	}

	// Create OTLP exporter for logs
	logExporter, err := otlploghttp.New(
		context.Background(),
		otlploghttp.WithEndpoint(config.Monitoring.Otel.Host),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create OTLP log exporter")
	}

	// Create TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.App.Name),
		)),
	)

	// Create LoggerProvider
	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
		log.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.App.Name),
		)),
	)

	// Add otellogrus hook to Logrus with LoggerProvider
	logger.AddHook(otellogrus.NewHook(config.App.Name, otellogrus.WithLoggerProvider(lp)))

	// Set the global tracer provider
	otel.SetTracerProvider(tp)

	return &Monitoring{
		tracerProvider: tp,
		loggerProvider: lp,
	}
}

func (m *Monitoring) Shutdown() error {
	// Shutdown both TracerProvider and LoggerProvider
	if err := m.tracerProvider.Shutdown(context.Background()); err != nil {
		return err
	}
	if err := m.loggerProvider.Shutdown(context.Background()); err != nil {
		return err
	}
	return nil
}
