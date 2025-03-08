package test

import (
	"go-starter-template/internal/config"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	config := config.NewViper()

	require.Equal(t, 10, config.GetInt("database.pool.idle"))
	require.Equal(t, 100, config.GetInt("database.pool.max"))
	require.Equal(t, 300, config.GetInt("database.pool.lifetime"))

	require.Equal(t, 6, config.GetInt("log.level"))

	require.Equal(t, "go-starter-template", config.GetString("app.name"))
	require.Equal(t, "secret", config.GetString("jwt.secret"))
	require.Equal(t, "refresh_secret", config.GetString("jwt.refresh_secret"))
}
