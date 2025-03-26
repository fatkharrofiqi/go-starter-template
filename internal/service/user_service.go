package service

import (
	"context"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/dto/converter"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/apperrors"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserService struct {
	DB             *gorm.DB
	UserRepository *repository.UserRepository
	Log            *logrus.Logger
}

func NewUserService(db *gorm.DB, userRepository *repository.UserRepository, logrus *logrus.Logger) *UserService {
	return &UserService{db, userRepository, logrus}
}

// GetUser retrieves a user by UUID.
func (s *UserService) GetUser(ctx context.Context, uuid string) (*dto.UserResponse, error) {
	user := new(model.User)
	if err := s.UserRepository.FindByUUID(s.DB, user, uuid); err != nil {
		s.Log.WithError(err).Warn("Failed to find user by UUID")
		return nil, apperrors.ErrUserNotFound
	}

	return &dto.UserResponse{
		UUID:      user.UUID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}, nil
}

// Search retrieves users based on search criteria.
func (s *UserService) Search(ctx context.Context, request *dto.SearchUserRequest) ([]*dto.UserResponse, int64, error) {
	users, total, err := s.UserRepository.Search(s.DB, request)
	if err != nil {
		s.Log.WithError(err).Error("Error retrieving users")
		return nil, 0, apperrors.ErrUserSearchFailed
	}

	responses := make([]*dto.UserResponse, len(users))
	for i, user := range users {
		responses[i] = converter.UserToResponse(&user)
	}

	return responses, total, nil
}
