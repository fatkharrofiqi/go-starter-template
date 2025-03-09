package service

import (
	"context"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/jwtutil"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	DB             *gorm.DB
	UserRepository *repository.UserRepository
	Viper          *viper.Viper
	Log            *logrus.Logger
}

func NewAuthService(db *gorm.DB, userRepository *repository.UserRepository, viper *viper.Viper, log *logrus.Logger) *AuthService {
	return &AuthService{db, userRepository, viper, log}
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	user, err := s.UserRepository.FindByEmail(tx, req.Email)
	if err != nil {
		s.Log.Warnf("Failed to find user by email : %v", err)
		return nil, fiber.NewError(fiber.ErrUnauthorized.Code, err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		s.Log.Warnf("Failed commit transaction : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.Log.Warnf("Failed to compare user password with bcrypt hash : %v", err)
		return nil, fiber.NewError(fiber.ErrUnauthorized.Code, err.Error())
	}

	accessToken, err := jwtutil.GenerateAccessToken(user.UUID, s.Viper.GetString("jwt.secret"))
	if err != nil {
		s.Log.Warnf("Failed generate access token : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	refreshToken, err := jwtutil.GenerateRefreshToken(user.UUID, s.Viper.GetString("jwt.refresh_secret"))
	if err != nil {
		s.Log.Warnf("Failed generate access token : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*model.User, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	found, err := s.UserRepository.CountByEmail(tx, req.Email)
	if err != nil {
		s.Log.Warnf("Failed count user from database : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	if found > 0 {
		s.Log.Warnf("User already exists")
		return nil, fiber.NewError(fiber.ErrConflict.Code, "User already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.Log.Warnf("Failed to generate bcrypt hashed password : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	user := &model.User{
		UUID:     uuid.New().String(),
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
	}

	if err := s.UserRepository.Repository.Create(tx, user); err != nil {
		s.Log.Warnf("Failed create user : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		s.Log.Warnf("Failed commit transaction : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	return user, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	claims, err := jwtutil.ValidateToken(refreshToken, s.Viper.GetString("jwt.refresh_secret"))
	if err != nil {
		s.Log.Warnf("Failed validate token : %v", err)
		return nil, fiber.NewError(fiber.ErrUnauthorized.Code, err.Error())
	}

	accessToken, err := jwtutil.GenerateAccessToken(claims.UUID, s.Viper.GetString("jwt.secret"))
	if err != nil {
		s.Log.Warnf("Failed generate access token : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
