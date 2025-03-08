package service

import (
	"context"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"

	"gorm.io/gorm"
)

type UserService struct {
	DB             *gorm.DB
	UserRepository *repository.UserRepository
}

func NewUserService(db *gorm.DB, userRepository *repository.UserRepository) *UserService {
	return &UserService{db, userRepository}
}

func (c *UserService) GetUser(ctx context.Context, uid string) (user *model.User, err error) {
	tx := c.DB.WithContext(ctx).Begin()
	user, err = c.UserRepository.FindByUID(tx, uid)
	if err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}
