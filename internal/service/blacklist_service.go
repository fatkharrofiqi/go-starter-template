package service

import (
	"context"
	"go-starter-template/internal/constant"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/errcode"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type BlacklistService struct {
	log                 *logrus.Logger
	jwtService          *JwtService
	blacklistRepository repository.TokenBlacklistRepository
	tracer              trace.Tracer
}

func NewBlacklistService(log *logrus.Logger, jwtService *JwtService, repo repository.TokenBlacklistRepository) *BlacklistService {
	return &BlacklistService{log, jwtService, repo, otel.Tracer("BlacklistService")}
}

func (b *BlacklistService) IsTokenBlacklisted(ctx context.Context, token string, tokenType constant.TokenType) error {
	spanCtx, span := b.tracer.Start(ctx, "BlacklistService.IsTokenBlacklisted")
	defer span.End()

	logger := b.log.WithContext(spanCtx)

	_, hashSpan := b.tracer.Start(spanCtx, "GenerateTokenHash")
	tokenHash := b.jwtService.GenerateTokenHash(token)
	hashSpan.End()

	_, checkSpan := b.tracer.Start(spanCtx, "CheckTokenBlacklistInRedis")
	blacklisted, err := b.blacklistRepository.IsBlacklisted(tokenHash, tokenType)
	checkSpan.End()
	if err != nil {
		logger.WithError(err).Error("failed to get token from Redis")
		return errcode.ErrRedisGet
	}

	if blacklisted {
		logger.Warn("token is already blacklisted")
		return errcode.ErrUnauthorized
	}

	return nil
}

func (b *BlacklistService) Add(ctx context.Context, token string, tokenType constant.TokenType) error {
	spanCtx, span := b.tracer.Start(ctx, "BlacklistService.Add")
	defer span.End()

	logger := b.log.WithContext(spanCtx)

	_, hashSpan := b.tracer.Start(spanCtx, "GenerateTokenHash")
	tokenHash := b.jwtService.GenerateTokenHash(token)
	hashSpan.End()

	parseCtx, parseSpan := b.tracer.Start(spanCtx, "ParseTokenClaims")
	claims, err := b.jwtService.ParseTokenClaims(parseCtx, token, tokenType)
	parseSpan.End()
	if err != nil {
		logger.WithError(err).Warn("could not parse token claims; fallback to default TTL")
		// Add fallback TTL to Redis
		_, fallbackSpan := b.tracer.Start(parseCtx, "AddFallbackTTLToRedis")
		err = b.blacklistRepository.Add(tokenHash, tokenType, 24*time.Hour)
		fallbackSpan.End()
		if err != nil {
			logger.WithError(err).Error("failed to store fallback token in Redis")
			return errcode.ErrRedisSet
		}
		return nil
	}

	_, ttlSpan := b.tracer.Start(spanCtx, "CalculateTTL")
	ttl := time.Until(claims.ExpiresAt.Time)
	ttlSpan.End()
	if ttl <= 0 {
		logger.Info("token is expired; skipping blacklist")
		return nil
	}

	_, addSpan := b.tracer.Start(spanCtx, "AddTokenHashToRedisWithTTL")
	err = b.blacklistRepository.Add(tokenHash, tokenType, ttl)
	addSpan.End()
	if err != nil {
		logger.WithError(err).Error("failed to store token in redis")
		return errcode.ErrRedisSet
	}

	return nil
}
