package service

import (
	"context"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/apperrors"

	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	DB               *gorm.DB
	JwtService       *JwtService
	UserRepository   *repository.UserRepository
	Logger           *logrus.Logger
	BlacklistService *BlacklistService
	Tracer           trace.Tracer
}

func NewAuthService(db *gorm.DB, jwtService *JwtService, userRepo *repository.UserRepository, blacklistService *BlacklistService, logger *logrus.Logger) *AuthService {
	return &AuthService{
		DB:               db,
		JwtService:       jwtService,
		UserRepository:   userRepo,
		Logger:           logger,
		BlacklistService: blacklistService,
		Tracer:           otel.Tracer("AuthService"),
	}
}

// Login authenticates a user and returns JWT tokens.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (string, string, error) {
	userContext, span := s.Tracer.Start(ctx, "Login")
	defer span.End()

	user := new(model.User)
	err := s.UserRepository.FindByEmail(userContext, user, req.Email)
	if err != nil {
		s.Logger.WithError(err).Error("User not found during login")
		return "", "", apperrors.ErrInvalidEmailOrPassword
	}

	// Validate password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.Logger.WithError(err).Error("Invalid password attempt")
		return "", "", apperrors.ErrInvalidEmailOrPassword
	}

	// Generate JWT tokens
	accessToken, err := s.JwtService.GenerateAccessToken(user.UUID)
	if err != nil {
		s.Logger.WithError(err).Error("Error generating access token")
		return "", "", apperrors.ErrAccessTokenGeneration
	}

	refreshToken, err := s.JwtService.GenerateRefreshToken(user.UUID)
	if err != nil {
		s.Logger.WithError(err).Error("Error generating refresh token")
		return "", "", apperrors.ErrRefreshTokenGeneration
	}

	return accessToken, refreshToken, nil
}

// Register creates a new user with a hashed password.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error) {
	userContext, span := s.Tracer.Start(ctx, "Register")
	defer span.End()

	tx := s.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			s.Logger.Error("Transaction panic recovered")
		}
	}()

	// Add transaction to context
	txContext := context.WithValue(userContext, repository.TxKey, tx)

	// Check if user already exists
	existingUserCount, err := s.UserRepository.CountByEmail(txContext, req.Email)
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

	if err := s.UserRepository.Create(txContext, &user); err != nil {
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
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	_, span := s.Tracer.Start(ctx, "RefreshToken")
	defer span.End()

	err := s.BlacklistService.IsTokenBlacklisted(refreshToken)
	if err != nil {
		log.Warn("already logout")
		return "", "", apperrors.ErrUnauthorized
	}

	claims, err := s.JwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		s.Logger.WithError(err).Error("Invalid refresh token")
		return "", "", apperrors.ErrInvalidToken
	}

	// Generate new access token
	accessToken, err := s.JwtService.GenerateAccessToken(claims.UUID)
	if err != nil {
		s.Logger.WithError(err).Error("Error generating new access token")
		return "", "", apperrors.ErrAccessTokenGeneration
	}

	// ROTATION: Generate new refresh token
	newRefreshToken, err := s.JwtService.GenerateRefreshToken(claims.UUID)
	if err != nil {
		s.Logger.WithError(err).Error("Error generating new refresh token")
		return "", "", apperrors.ErrRefreshTokenGeneration
	}

	// ROTATION: Blacklist old refresh token
	if err := s.BlacklistService.Add(refreshToken); err != nil {
		s.Logger.WithError(err).Error("Failed to blacklist old refresh token")
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	_, span := s.Tracer.Start(ctx, "Logout")
	defer span.End()

	// Add the token to a blacklist or revocation list.
	if err := s.BlacklistService.Add(accessToken); err != nil {
		s.Logger.WithError(err).Error("Failed to invalidate access token to redis")
		return err
	}

	if err := s.BlacklistService.Add(refreshToken); err != nil {
		s.Logger.WithError(err).Error("Failed to invalidate refresh token to redis")
		return err
	}

	return nil
}
