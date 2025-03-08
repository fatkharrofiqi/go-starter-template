package repository

import (
	"go-starter-template/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	Repository[model.User]
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) FindByEmail(tx *gorm.DB, email string) (*model.User, error) {
	var user model.User
	if err := tx.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByUID(tx *gorm.DB, uid string) (*model.User, error) {
	var user model.User
	if err := tx.Where("uid = ?", uid).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
