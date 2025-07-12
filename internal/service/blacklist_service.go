package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

func (b *BlacklistService) IsTokenBlacklisted(ctx context.Context, token string) error {
	spanCtx, span := b.tracer.Start(ctx, "IsTokenBlacklisted")
	defer span.End()

	tokenHash := b.generateTokenHash(token)
	logout, err := b.blacklistRepository.IsBlacklisted(tokenHash)
	if err != nil {
		b.log.WithContext(spanCtx).WithError(err).Error("Get redis failed")
		return errcode.ErrRedisGet
	}

	if logout {
		b.log.WithContext(spanCtx).Error("Already logout")
		return errcode.ErrUnauthorized
	}

	return nil
}

func (b *BlacklistService) Add(ctx context.Context, token string) error {
	spanCtx, span := b.tracer.Start(ctx, "Add")
	defer span.End()

	// Generate hash for security & efficiency
	tokenHash := b.generateTokenHash(token)

	// Parse token to get expiration time
	claims, err := b.parseTokenClaims(spanCtx, token)
	if err != nil {
		// if parse failed, set default TTL
		b.log.WithContext(spanCtx).WithError(err).Error("Failed to parse claims")
		return b.blacklistRepository.Add(tokenHash, 24*time.Hour)
	}

	// Set TTL based on expiration time token
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		// Token is expired, no need to blacklist
		b.log.WithContext(spanCtx).Info("Token is expired, no need to blacklist")
		return nil
	}

	if err := b.blacklistRepository.Add(tokenHash, ttl); err != nil {
		b.log.WithContext(spanCtx).Info("Set redis failed")
		return errcode.ErrRedisSet
	}

	return nil
}

// Generate SHA256 hash
func (b *BlacklistService) generateTokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Parse token to get claims
func (b *BlacklistService) parseTokenClaims(ctx context.Context, token string) (*Claims, error) {
	spanCtx, span := b.tracer.Start(ctx, "parseTokenClaims")
	defer span.End()

	return b.jwtService.ValidateRefreshToken(spanCtx, token)
}
