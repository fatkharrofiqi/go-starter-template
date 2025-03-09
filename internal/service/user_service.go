package service

import (
	"context"
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

func (c *UserService) GetUser(ctx context.Context, uuid string) (*model.User, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	user, err := c.UserRepository.FindByUUID(tx, uuid)
	if err != nil {
		c.Log.Warnf("Failed to find user by UUID : %v", err)
		return nil, fiber.NewError(fiber.ErrNotFound.Code, err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %v", err)
		return nil, fiber.NewError(fiber.ErrInternalServerError.Code, err.Error())
	}

	return user, err
}
