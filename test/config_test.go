package test

import (
	"go-starter-template/internal/config/env"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	config := env.NewConfig()

	require.Equal(t, 10, config.Database.Pool.Idle)
	require.Equal(t, 100, config.Database.Pool.Max)
	require.Equal(t, 300, config.Database.Pool.Lifetime)

	require.Equal(t, 6, config.Log.Level)

	require.Equal(t, "go-starter-template", config.App.Name)
	require.Equal(t, "secret", config.JWT.Secret)
	require.Equal(t, "refresh_secret", config.JWT.RefreshSecret)
}
