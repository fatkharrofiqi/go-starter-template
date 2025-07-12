package service

import (
	"context"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/errcode"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db               *gorm.DB
	jwtService       *JwtService
	userRepository   *repository.UserRepository
	logger           *logrus.Logger
	blacklistService *BlacklistService
	tracer           trace.Tracer
}

func NewAuthService(db *gorm.DB, jwtService *JwtService, userRepo *repository.UserRepository, blacklistService *BlacklistService, logger *logrus.Logger) *AuthService {
	return &AuthService{db, jwtService, userRepo, logger, blacklistService, otel.Tracer("AuthService")}
}

// Login authenticates a user and returns JWT tokens.
func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest) (string, string, error) {
	spanCtx, span := s.tracer.Start(ctx, "Login")
	defer span.End()

	user := new(model.User)
	err := s.userRepository.FindByEmail(spanCtx, user, req.Email)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("User not found during login")
		return "", "", errcode.ErrInvalidEmailOrPassword
	}

	// Validate password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Invalid password attempt")
		return "", "", errcode.ErrInvalidEmailOrPassword
	}

	// Generate JWT tokens
	accessToken, err := s.jwtService.GenerateAccessToken(spanCtx, user.UUID)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Error generating access token")
		return "", "", errcode.ErrAccessTokenGeneration
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(spanCtx, user.UUID)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Error generating refresh token")
		return "", "", errcode.ErrRefreshTokenGeneration
	}

	return accessToken, refreshToken, nil
}

// Register creates a new user with a hashed password.
func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.UserResponse, error) {
	spanCtx, span := s.tracer.Start(ctx, "Register")
	defer span.End()

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			s.logger.WithContext(spanCtx).Error("Transaction panic recovered")
		}
	}()

	// Add transaction to context
	txContext := context.WithValue(spanCtx, repository.TxKey, tx)

	// Check if user already exists
	existingUserCount, err := s.userRepository.CountByEmail(txContext, req.Email)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Database error checking existing user")
		return nil, errcode.ErrDatabaseError
	}

	if existingUserCount > 0 {
		s.logger.WithContext(spanCtx).Warn("Attempt to register an already existing email")
		return nil, errcode.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Failed to hash password")
		return nil, errcode.ErrPasswordEncryption
	}

	user := model.User{
		UUID:     uuid.New().String(),
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
	}

	if err := s.userRepository.Create(txContext, &user); err != nil {
		tx.Rollback()
		s.logger.WithContext(spanCtx).WithError(err).Error("Error creating user")
		return nil, errcode.ErrUserCreationFailed
	}

	if err := tx.Commit().Error; err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Transaction commit failed")
		return nil, errcode.ErrDatabaseTransaction
	}

	return &dto.UserResponse{
		UUID:      user.UUID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}, nil
}

// RefreshToken generates a new access token using a valid refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	spanCtx, span := s.tracer.Start(ctx, "RefreshToken")
	defer span.End()

	err := s.blacklistService.IsTokenBlacklisted(spanCtx, refreshToken)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Already logout")
		return "", "", errcode.ErrUnauthorized
	}

	claims, err := s.jwtService.ValidateRefreshToken(spanCtx, refreshToken)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Invalid refresh token")
		return "", "", errcode.ErrInvalidToken
	}

	// Generate new access token
	accessToken, err := s.jwtService.GenerateAccessToken(spanCtx, claims.UUID)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Error generating new access token")
		return "", "", errcode.ErrAccessTokenGeneration
	}

	// ROTATION: Generate new refresh token
	newRefreshToken, err := s.jwtService.GenerateRefreshToken(spanCtx, claims.UUID)
	if err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Error generating new refresh token")
		return "", "", errcode.ErrRefreshTokenGeneration
	}

	// ROTATION: Blacklist old refresh token
	if err := s.blacklistService.Add(spanCtx, refreshToken); err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Failed to blacklist old refresh token")
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	spanCtx, span := s.tracer.Start(ctx, "Logout")
	defer span.End()

	// Add the token to a blacklist or revocation list.
	if err := s.blacklistService.Add(spanCtx, accessToken); err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Failed to invalidate access token to redis")
		return err
	}

	if err := s.blacklistService.Add(spanCtx, refreshToken); err != nil {
		s.logger.WithContext(spanCtx).WithError(err).Error("Failed to invalidate refresh token to redis")
		return err
	}

	return nil
}
