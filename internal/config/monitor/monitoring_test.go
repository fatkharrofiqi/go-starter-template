package monitor

import (
    "testing"
    "bytes"
    "fmt"

    "go-starter-template/internal/config/env"

    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

// Table-driven tests for NewMonitoring
func TestNewMonitoring_TableDriven(t *testing.T) {
    cases := []struct {
        name    string
        appName string
        host    string
    }{
        {name: "valid host and app", appName: "test-app", host: "localhost:4318"},
        {name: "empty host", appName: "test-app", host: ""},
        {name: "empty app name", appName: "", host: "localhost:4318"},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            logger := logrus.New()

            // Ensure hooks start empty
            for level := logrus.PanicLevel; level <= logrus.TraceLevel; level++ {
                assert.Len(t, logger.Hooks[level], 0)
            }

            cfg := &env.Config{}
            cfg.App.Name = tc.appName
            cfg.Monitoring.Otel.Host = tc.host

            m := NewMonitoring(logger, cfg)
            require.NotNil(t, m)
            require.NotNil(t, m.tracerProvider)
            require.NotNil(t, m.loggerProvider)

            // Global tracer provider should be set to the one created
            globalTP := otel.GetTracerProvider()
            assert.Equal(t, m.tracerProvider, globalTP)

            // otellogrus hook should be added across levels
            added := 0
            for level := logrus.PanicLevel; level <= logrus.TraceLevel; level++ {
                added += len(logger.Hooks[level])
            }
            assert.Greater(t, added, 0)
        })
    }
}

// Probe which endpoints cause trace exporter creation to fail (helper to guide fatal tests)
func TestProbeTraceExporterEndpointBehavior(t *testing.T) {
    cases := []string{"localhost:4318", "http://", "://", "bad:endpoint", "127.0.0.1:bad"}
    for _, ep := range cases {
        t.Run(fmt.Sprintf("ep=%s", ep), func(t *testing.T) {
            _, err := otlptrace.New(
                t.Context(),
                otlptracehttp.NewClient(
                    otlptracehttp.WithEndpoint(ep),
                    otlptracehttp.WithInsecure(),
                ),
            )
            if err != nil {
                t.Logf("trace exporter error for endpoint '%s': %v", ep, err)
            } else {
                t.Logf("trace exporter created successfully for endpoint '%s'", ep)
            }
        })
    }
}

// Table-driven tests for Shutdown
func TestMonitoringShutdown_TableDriven(t *testing.T) {
    cases := []struct {
        name string
        host string
    }{
        {name: "shutdown with valid host", host: "localhost:4318"},
        {name: "shutdown with empty host", host: ""},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            logger := logrus.New()
            cfg := &env.Config{}
            cfg.App.Name = "test-app"
            cfg.Monitoring.Otel.Host = tc.host

            m := NewMonitoring(logger, cfg)
            err := m.Shutdown()
            assert.NoError(t, err)
        })
    }
}

// Attempt to cover fatal path when exporter creation fails
func TestNewMonitoring_InvalidEndpoint_TriggersFatal(t *testing.T) {
    cases := []struct{
        name string
        host string
    }{
        {name: "clearly invalid", host: "://"},
        {name: "bad scheme", host: "http://"},
        {name: "invalid port string", host: "localhost:http"},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            logger := logrus.New()
            buf := &bytes.Buffer{}
            logger.SetOutput(buf)
            // Override ExitFunc to panic so we can catch Fatal without exiting
            logger.ExitFunc = func(code int) { panic("fatal exit") }

            cfg := &env.Config{}
            cfg.App.Name = "test-app"
            cfg.Monitoring.Otel.Host = tc.host

            assert.Panics(t, func() { _ = NewMonitoring(logger, cfg) })
            // Ensure a fatal OTLP exporter creation message was logged
            assert.Contains(t, buf.String(), "Failed to create OTLP")
        })
    }
}