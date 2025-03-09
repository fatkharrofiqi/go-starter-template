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

func (r *UserRepository) CountByEmail(tx *gorm.DB, email string) (int64, error) {
	var total int64
	var user model.User
	err := tx.Model(&user).Where("email = ?", email).Count(&total).Error
	return total, err
}

func (r *UserRepository) FindByEmail(tx *gorm.DB, email string) (*model.User, error) {
	var user model.User
	if err := tx.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByUUID(tx *gorm.DB, uuid string) (*model.User, error) {
	var user model.User
	if err := tx.Where("uuid = ?", uuid).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
