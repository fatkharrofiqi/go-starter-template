package service

import (
	"context"
	"go-starter-template/internal/constant"
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
func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest) (accessToken string, refreshToken string, err error) {
	spanCtx, span := s.tracer.Start(ctx, "AuthService.Login")
	defer span.End()

	logger := s.logger.WithContext(spanCtx)

	user := new(model.User)
	if err := s.userRepository.FindByEmail(spanCtx, user, req.Email); err != nil {
		logger.WithError(err).Error("User not found during login")
		return "", "", errcode.ErrInvalidEmailOrPassword
	}

	// Validate password
	_, passwordSpan := s.tracer.Start(spanCtx, "CompareHashPassword")
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		passwordSpan.End()
		logger.WithError(err).Error("Invalid password attempt")
		return "", "", errcode.ErrInvalidEmailOrPassword
	}
	passwordSpan.End()

	// Generate JWT tokens
	if accessToken, err = s.jwtService.GenerateAccessToken(spanCtx, user.UUID); err != nil {
		logger.WithError(err).Error("Error generating access token")
		return "", "", errcode.ErrAccessTokenGeneration
	}

	if refreshToken, err = s.jwtService.GenerateRefreshToken(spanCtx, user.UUID); err != nil {
		logger.WithError(err).Error("Error generating refresh token")
		return "", "", errcode.ErrRefreshTokenGeneration
	}

	return accessToken, refreshToken, nil
}

// Register creates a new user with a hashed password.
func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.UserResponse, error) {
	spanCtx, span := s.tracer.Start(ctx, "AuthService.Register")
	defer span.End()

	logger := s.logger.WithContext(spanCtx)
	tx := s.db.Begin()
	txCtx := context.WithValue(spanCtx, repository.TxKey, tx)

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.WithField("panic", r).Error("Recovered from panic during registration")
		}
	}()

	existingUserCount, err := s.userRepository.CountByEmail(txCtx, req.Email)
	if err != nil {
		logger.WithError(err).Error("Database error checking existing user")
		return nil, errcode.ErrDatabaseError
	}
	if existingUserCount > 0 {
		logger.Warn("Attempt to register an already existing email")
		return nil, errcode.ErrUserAlreadyExists
	}

	_, hashSpan := s.tracer.Start(spanCtx, "HashPassword")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	hashSpan.End()
	if err != nil {
		logger.WithError(err).Error("Failed to hash password")
		return nil, errcode.ErrPasswordEncryption
	}

	user := model.User{
		UUID:     uuid.New().String(),
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
	}

	if err := s.userRepository.Create(txCtx, &user); err != nil {
		tx.Rollback()
		logger.WithError(err).Error("Error creating user")
		return nil, errcode.ErrUserCreationFailed
	}

	if err := tx.Commit().Error; err != nil {
		logger.WithError(err).Error("Transaction commit failed")
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
	spanCtx, span := s.tracer.Start(ctx, "AuthService.RefreshToken")
	defer span.End()
	logger := s.logger.WithContext(spanCtx)

	err := s.blacklistService.IsTokenBlacklisted(spanCtx, refreshToken, constant.TokenTypeRefresh)
	if err != nil {
		logger.WithError(err).Error("Already logout")
		return "", "", errcode.ErrUnauthorized
	}

	claims, err := s.jwtService.ValidateRefreshToken(spanCtx, refreshToken)
	if err != nil {
		logger.WithError(err).Error("Invalid refresh token")
		return "", "", errcode.ErrInvalidToken
	}

	accessToken, err := s.jwtService.GenerateAccessToken(spanCtx, claims.UUID)
	if err != nil {
		logger.WithError(err).Error("Error generating new access token")
		return "", "", errcode.ErrAccessTokenGeneration
	}

	newRefreshToken, err := s.jwtService.GenerateRefreshToken(spanCtx, claims.UUID)
	if err != nil {
		logger.WithError(err).Error("Error generating new refresh token")
		return "", "", errcode.ErrRefreshTokenGeneration
	}

	if err := s.blacklistService.Add(spanCtx, refreshToken, constant.TokenTypeRefresh); err != nil {
		logger.WithError(err).Error("Failed to blacklist old refresh token")
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

// Logout invalidates access and refresh tokens.
func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	spanCtx, span := s.tracer.Start(ctx, "AuthService.Logout")
	defer span.End()
	logger := s.logger.WithContext(spanCtx)

	if err := s.blacklistService.Add(spanCtx, accessToken, constant.TokenTypeAccess); err != nil {
		logger.WithError(err).Error("Failed to invalidate access token to redis")
		return err
	}

	if err := s.blacklistService.Add(spanCtx, refreshToken, constant.TokenTypeRefresh); err != nil {
		logger.WithError(err).Error("Failed to invalidate refresh token to redis")
		return err
	}

	return nil
}
