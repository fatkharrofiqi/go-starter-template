package service

import (
	"context"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/dto/converter"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"

	"github.com/gofiber/fiber/v2"
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

func (c *UserService) GetUser(ctx context.Context, uuid string) (*dto.UserResponse, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	user := new(model.User)
	if err := c.UserRepository.FindByUUID(tx, user, uuid); err != nil {
		c.Log.WithError(err).Error("error find user by uuid")
		return nil, fiber.NewError(fiber.ErrNotFound.Code, err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.WithError(err).Error("error commit transaction")
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	return &dto.UserResponse{
		UUID:      user.UUID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}, nil
}

func (c *UserService) Search(ctx context.Context, request *dto.SearchUserRequest) ([]dto.UserResponse, int64, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	users, total, err := c.UserRepository.Search(tx, request)
	if err != nil {
		c.Log.WithError(err).Error("error getting users")
		return nil, 0, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.WithError(err).Error("failed to commit transaction")
		return nil, 0, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	responses := make([]dto.UserResponse, len(users))
	for i, user := range users {
		responses[i] = *converter.UserToResponse(&user)
	}

	return responses, total, nil
}
