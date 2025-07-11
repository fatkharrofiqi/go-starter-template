package service

import (
	"crypto/sha256"
	"encoding/hex"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/errcode"
	"time"
)

type BlacklistService struct {
	JwtService          *JwtService
	BlacklistRepository repository.TokenBlacklistRepository
}

func NewBlacklistService(jwtService *JwtService, repo repository.TokenBlacklistRepository) *BlacklistService {
	return &BlacklistService{jwtService, repo}
}

func (b *BlacklistService) IsTokenBlacklisted(token string) error {
	tokenHash := b.generateTokenHash(token)
	logout, err := b.BlacklistRepository.IsBlacklisted(tokenHash)
	if err != nil {
		return errcode.ErrRedisGet
	}

	if logout {
		return errcode.ErrUnauthorized
	}

	return nil
}

func (b *BlacklistService) Add(token string) error {
	// Generate hash for security & efficiency
	tokenHash := b.generateTokenHash(token)

	// Parse token to get expiration time
	claims, err := b.parseTokenClaims(token)
	if err != nil {
		// if parse failed, set default TTL
		return b.BlacklistRepository.Add(tokenHash, 24*time.Hour)
	}

	// Set TTL based on expiration time token
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		// Token is expired, no need to blacklist
		return nil
	}

	if err := b.BlacklistRepository.Add(tokenHash, ttl); err != nil {
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
func (b *BlacklistService) parseTokenClaims(token string) (*Claims, error) {
	return b.JwtService.ValidateRefreshToken(token)
}
