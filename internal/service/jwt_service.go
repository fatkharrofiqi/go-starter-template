package service

import (
	"errors"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/apperrors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UUID string `json:"uuid"`
	Type string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type JwtService struct {
	AccessTokenExpiration    time.Duration
	RefreshTokenExpiration   time.Duration
	TokenBlacklistRepository repository.TokenBlacklistRepository
	accessTokenSecret        string
	refreshTokenSecret       string
}

func NewJwtService(config *env.Config, tokenBlacklistRepository repository.TokenBlacklistRepository) *JwtService {
	return &JwtService{
		AccessTokenExpiration:    config.JWT.AccessTokenExpiration * time.Second,
		RefreshTokenExpiration:   config.JWT.RefreshTokenExpiration * time.Second,
		TokenBlacklistRepository: tokenBlacklistRepository,
		accessTokenSecret:        config.JWT.Secret,
		refreshTokenSecret:       config.JWT.RefreshSecret,
	}
}

// GenerateAccessToken creates a short-lived JWT access token
func (j *JwtService) GenerateAccessToken(uuid string) (string, error) {
	claims := Claims{
		UUID: uuid,
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.AccessTokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.accessTokenSecret))
}

// GenerateRefreshToken creates a long-lived JWT refresh token
func (j *JwtService) GenerateRefreshToken(uuid string) (string, error) {
	claims := Claims{
		UUID: uuid,
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.RefreshTokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.refreshTokenSecret))
}

func (j *JwtService) ValidateAccessToken(token string) (*Claims, error) {
	return j.validateToken(token, j.accessTokenSecret)
}

func (j *JwtService) ValidateRefreshToken(token string) (*Claims, error) {
	return j.validateToken(token, j.refreshTokenSecret)
}

// ValidateToken verifies a JWT token and returns the claims if valid
func (j *JwtService) validateToken(tokenString string, secretKey string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, apperrors.ErrInvalidToken
	}

	return claims, nil
}
