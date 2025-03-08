package service

import (
	"context"
	"fmt"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/errwrap"
	"go-starter-template/internal/utils/jwtutil"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	DB             *gorm.DB
	UserRepository *repository.UserRepository
	Viper          *viper.Viper
}

func NewAuthService(db *gorm.DB, userRepository *repository.UserRepository, viper *viper.Viper) *AuthService {
	return &AuthService{db, userRepository, viper}
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	user, err := s.UserRepository.FindByEmail(tx, req.Email)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("transaction commit failed: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, err
	}

	accessToken, err := jwtutil.GenerateAccessToken(user.UID, s.Viper.GetString("jwt.secret"))
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwtutil.GenerateRefreshToken(user.UID, s.Viper.GetString("jwt.refresh_secret"))
	if err != nil {
		return nil, err
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) error {
	tx := s.DB.WithContext(ctx).Begin()

	// Ensure rollback in case of any error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Hash password securely
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Check if email already exists
	found, _ := s.UserRepository.FindByEmail(tx, req.Email)

	if found != nil {
		tx.Rollback()
		return errwrap.WrapError(errwrap.ErrDataExists, fmt.Sprintf("user %s already exist", found.Email))
	}

	// Create user model
	user := &model.User{
		UID:      uuid.New().String(),
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
	}

	// Insert user into DB
	if err := s.UserRepository.Repository.Create(tx, user); err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	claims, err := jwtutil.ValidateToken(refreshToken, s.Viper.GetString("jwt.refresh_secret"))
	if err != nil {
		return nil, err
	}

	accessToken, err := jwtutil.GenerateAccessToken(claims.UID, s.Viper.GetString("jwt.secret"))
	if err != nil {
		return nil, err
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
