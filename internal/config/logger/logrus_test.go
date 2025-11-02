package logger

import (
    "testing"

    "go-starter-template/internal/config/env"

    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/require"
)

func TestNewLogger_FormatterAndLevel(t *testing.T) {
    cfg := &env.Config{}
    cfg.Log.Level = int(logrus.InfoLevel)

    log := NewLogger(cfg)
    require.NotNil(t, log)

    // Verify level is set from config
    require.Equal(t, logrus.InfoLevel, log.Level)

    // Verify formatter settings
    tf, ok := log.Formatter.(*logrus.TextFormatter)
    require.True(t, ok, "expected TextFormatter")
    require.True(t, tf.ForceColors)
    require.True(t, tf.FullTimestamp)
    require.Equal(t, "2006-01-02 15:04:05", tf.TimestampFormat)
}

func TestNewLogger_LevelMapping(t *testing.T) {
    levels := []logrus.Level{
        logrus.PanicLevel,
        logrus.FatalLevel,
        logrus.ErrorLevel,
        logrus.WarnLevel,
        logrus.InfoLevel,
        logrus.DebugLevel,
        logrus.TraceLevel,
    }

    for _, lv := range levels {
        cfg := &env.Config{}
        cfg.Log.Level = int(lv)
        log := NewLogger(cfg)
        require.Equal(t, lv, log.Level)
    }
}