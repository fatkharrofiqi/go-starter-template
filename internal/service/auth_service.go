package service

import (
	"context"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/apperrors"
	"go-starter-template/internal/utils/jwtutil"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	DB             *gorm.DB
	UserRepository *repository.UserRepository
	Config         *env.Config
	Logger         *logrus.Logger
	TokenBlacklist *repository.TokenBlacklist
	Tracer         trace.Tracer
}

func NewAuthService(db *gorm.DB, userRepo *repository.UserRepository, tokenBlacklist *repository.TokenBlacklist, config *env.Config, logger *logrus.Logger) *AuthService {
	return &AuthService{
		DB:             db,
		UserRepository: userRepo,
		Config:         config,
		Logger:         logger,
		TokenBlacklist: tokenBlacklist,
		Tracer:         otel.Tracer("AuthService"),
	}
}

// Login authenticates a user and returns JWT tokens.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error) {
	userContext, span := s.Tracer.Start(ctx, "Login")
	defer span.End()

	user := new(model.User)
	err := s.UserRepository.FindByEmail(s.DB.WithContext(userContext), user, req.Email)
	if err != nil {
		s.Logger.WithError(err).Error("User not found during login")
		return nil, apperrors.ErrInvalidEmailOrPassword
	}

	// Validate password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.Logger.WithError(err).Error("Invalid password attempt")
		return nil, apperrors.ErrInvalidEmailOrPassword
	}

	// Generate JWT tokens
	accessToken, err := jwtutil.GenerateAccessToken(user.UUID, s.Config.JWT.Secret)
	if err != nil {
		s.Logger.WithError(err).Error("Error generating access token")
		return nil, apperrors.ErrAccessTokenGeneration
	}

	refreshToken, err := jwtutil.GenerateRefreshToken(user.UUID, s.Config.JWT.RefreshSecret)
	if err != nil {
		s.Logger.WithError(err).Error("Error generating refresh token")
		return nil, apperrors.ErrRefreshTokenGeneration
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Register creates a new user with a hashed password.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error) {
	userContext, span := s.Tracer.Start(ctx, "Register")
	defer span.End()

	tx := s.DB.WithContext(userContext).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			s.Logger.Error("Transaction panic recovered")
		}
	}()

	// Check if user already exists
	existingUserCount, err := s.UserRepository.CountByEmail(tx, req.Email)
	if err != nil {
		s.Logger.WithError(err).Error("Database error checking existing user")
		return nil, apperrors.ErrDatabaseError
	}

	if existingUserCount > 0 {
		s.Logger.Warn("Attempt to register an already existing email")
		return nil, apperrors.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.WithError(err).Error("Failed to hash password")
		return nil, apperrors.ErrPasswordEncryption
	}

	user := model.User{
		UUID:     uuid.New().String(),
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
	}

	if err := s.UserRepository.Repository.Create(tx, &user); err != nil {
		tx.Rollback()
		s.Logger.WithError(err).Error("Error creating user")
		return nil, apperrors.ErrUserCreationFailed
	}

	if err := tx.Commit().Error; err != nil {
		s.Logger.WithError(err).Error("Transaction commit failed")
		return nil, apperrors.ErrDatabaseTransaction
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
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	_, span := s.Tracer.Start(ctx, "RefreshToken")
	defer span.End()

	if s.TokenBlacklist.IsBlacklisted(refreshToken) {
		s.Logger.Error("token is blacklisted")
		return nil, apperrors.ErrTokenBlacklisted
	}

	claims, err := jwtutil.ValidateToken(refreshToken, s.Config.JWT.RefreshSecret)
	if err != nil {
		s.Logger.WithError(err).Error("Invalid refresh token")
		return nil, apperrors.ErrInvalidToken
	}

	// Generate new access token
	accessToken, err := jwtutil.GenerateAccessToken(claims.UUID, s.Config.JWT.Secret)
	if err != nil {
		s.Logger.WithError(err).Error("Error generating new access token")
		return nil, apperrors.ErrAccessTokenGeneration
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // Reuse the same refresh token
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	_, span := s.Tracer.Start(ctx, "Logout")
	defer span.End()

	// Add the token to a blacklist or revocation list.
	if err := s.TokenBlacklist.Add(accessToken); err != nil {
		s.Logger.WithError(err).Error("Failed to invalidate token")
		return apperrors.ErrTokenInvalidation
	}

	if err := s.TokenBlacklist.Add(refreshToken); err != nil {
		s.Logger.WithError(err).Error("Failed to invalidate refresh token")
		return apperrors.ErrTokenInvalidation
	}

	return nil
}
