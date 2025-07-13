package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/constant"
	"go-starter-template/internal/utils/errcode"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Claims struct {
	UUID string `json:"uuid"`
	Type string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type JwtService struct {
	log    *logrus.Logger
	config *env.Config
	tracer trace.Tracer
}

func NewJwtService(log *logrus.Logger, config *env.Config) *JwtService {
	return &JwtService{log, config, otel.Tracer("JwtService")}
}

// GenerateAccessToken creates a short-lived JWT access token
func (j *JwtService) GenerateAccessToken(ctx context.Context, uuid string) (string, error) {
	_, span := j.tracer.Start(ctx, "JwtService.GenerateAccessToken")
	defer span.End()

	claims := Claims{
		UUID: uuid,
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.config.GetAccessTokenExpiration())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.GetAccessSecret()))
}

// GenerateRefreshToken creates a long-lived JWT refresh token
func (j *JwtService) GenerateRefreshToken(ctx context.Context, uuid string) (string, error) {
	_, span := j.tracer.Start(ctx, "JwtService.GenerateRefreshToken")
	defer span.End()

	claims := Claims{
		UUID: uuid,
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.config.GetRefreshTokenExpiration())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.GetRefreshSecret()))
}

func (j *JwtService) ValidateAccessToken(ctx context.Context, token string) (*Claims, error) {
	spanCtx, span := j.tracer.Start(ctx, "JwtService.ValidateAccessToken")
	defer span.End()

	return j.validateToken(spanCtx, token, j.config.GetAccessSecret())
}

func (j *JwtService) ValidateRefreshToken(ctx context.Context, token string) (*Claims, error) {
	spanCtx, span := j.tracer.Start(ctx, "JwtService.ValidateRefreshToken")
	defer span.End()

	return j.validateToken(spanCtx, token, j.config.GetRefreshSecret())
}

// ValidateToken verifies a JWT token and returns the claims if valid
func (j *JwtService) validateToken(ctx context.Context, tokenString string, secretKey string) (*Claims, error) {
	spanCtx, span := j.tracer.Start(ctx, "JwtService.validateToken")
	defer span.End()

	logger := j.log.WithContext(spanCtx)

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Error("Token method not match")
			return nil, errcode.ErrUnexpectedSignMethod
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		logger.WithError(err).Error("Failed to parse with claims")
		return nil, err
	}

	if !token.Valid {
		logger.WithError(err).Error("Token invalid")
		return nil, errcode.ErrInvalidToken
	}

	return claims, nil
}

// Generate SHA256 hash
func (j *JwtService) GenerateTokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Parse token to get claims
func (j *JwtService) ParseTokenClaims(ctx context.Context, token string, tokenType constant.TokenType) (*Claims, error) {
	spanCtx, span := j.tracer.Start(ctx, "JwtService.parseTokenClaims")
	defer span.End()

	switch tokenType {
	case constant.TokenTypeAccess:
		return j.ValidateAccessToken(spanCtx, token)
	case constant.TokenTypeRefresh:
		return j.ValidateRefreshToken(spanCtx, token)
	default:
		return nil, fmt.Errorf("unsupported token type: %s", tokenType)
	}
}
