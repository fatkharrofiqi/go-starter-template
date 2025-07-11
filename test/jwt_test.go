package test

import (
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/service"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupConfig() *env.Config {
	return env.NewConfig()
}

func setupConfigError() *env.Config {
	cfg := env.NewConfig()
	cfg.JWT.AccessTokenExpiration = -1 * time.Second
	cfg.JWT.RefreshTokenExpiration = -1 * time.Second
	return cfg
}

func setupService(cfg *env.Config) *service.JwtService {
	service := service.NewJwtService(cfg)
	return service
}

func TestGenerateAccessToken(t *testing.T) {
	service := setupService(setupConfig())
	uuid := "user123"
	token, err := service.GenerateAccessToken(uuid)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := service.ValidateAccessToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uuid, claims.UUID)
	assert.Equal(t, "access", claims.Type)
}

func TestGenerateRefreshToken(t *testing.T) {
	service := setupService(setupConfig())
	uuid := "user123"
	token, err := service.GenerateRefreshToken(uuid)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := service.ValidateRefreshToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uuid, claims.UUID)
	assert.Equal(t, "refresh", claims.Type)
}

func TestExpiredToken(t *testing.T) {
	service := setupService(setupConfigError())

	uuid := "user123"
	token, err := service.GenerateAccessToken(uuid)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	_, err = service.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestInvalidToken(t *testing.T) {
	service := setupService(setupConfig())
	invalidToken := "invalid.token.string"

	_, err := service.ValidateAccessToken(invalidToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is malformed")
}
