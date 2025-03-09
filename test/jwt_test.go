package test

import (
	"go-starter-template/internal/utils/jwtutil"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const secretKey = "test_secret"

func TestGenerateAccessToken(t *testing.T) {
	uuid := "user123"
	token, err := jwtutil.GenerateAccessToken(uuid, secretKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwtutil.ValidateToken(token, secretKey)
	assert.NoError(t, err)
	assert.Equal(t, uuid, claims.UUID)
	assert.Equal(t, "access", claims.Type)
}

func TestGenerateRefreshToken(t *testing.T) {
	uuid := "user123"
	token, err := jwtutil.GenerateRefreshToken(uuid, secretKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwtutil.ValidateToken(token, secretKey)
	assert.NoError(t, err)
	assert.Equal(t, uuid, claims.UUID)
	assert.Equal(t, "refresh", claims.Type)
}

func TestValidateToken(t *testing.T) {
	uuid := "user123"
	token, err := jwtutil.GenerateAccessToken(uuid, secretKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwtutil.ValidateToken(token, secretKey)
	assert.NoError(t, err)
	assert.Equal(t, uuid, claims.UUID)
	assert.Equal(t, "access", claims.Type)
}

func TestExpiredToken(t *testing.T) {
	// Create a token that expired 1 second ago
	claims := jwtutil.Claims{
		UUID: "user123",
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-10 * time.Second)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secretKey))

	_, err := jwtutil.ValidateToken(tokenString, secretKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestInvalidToken(t *testing.T) {
	invalidToken := "invalid.token.string"

	_, err := jwtutil.ValidateToken(invalidToken, secretKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is malformed")
}
