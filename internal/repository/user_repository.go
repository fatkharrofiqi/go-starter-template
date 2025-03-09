package repository

import (
	"go-starter-template/internal/dto"
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

func (r *UserRepository) FindByEmail(tx *gorm.DB, user *model.User, email string) error {
	return tx.Where("email = ?", email).First(&user).Error
}

func (r *UserRepository) FindByUUID(tx *gorm.DB, user *model.User, uuid string) error {
	return tx.Where("uuid = ?", uuid).First(&user).Error
}

func (r *UserRepository) Search(tx *gorm.DB, request *dto.SearchUserRequest) ([]model.User, int64, error) {
	var user []model.User
	if err := tx.Scopes(r.FilterUser(request)).Offset((request.Page - 1) * request.Size).Limit(request.Size).Find(&user).Error; err != nil {
		return nil, 0, err
	}

	var total int64 = 0
	if err := tx.Model(&model.User{}).Scopes(r.FilterUser(request)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return user, total, nil
}

func (r *UserRepository) FilterUser(request *dto.SearchUserRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if name := request.Name; name != "" {
			name = "%" + name + "%"
			tx = tx.Where("name LIKE ?", name)
		}

		if email := request.Email; email != "" {
			email = "%" + email + "%"
			tx = tx.Where("email LIKE ?", email)
		}

		return tx
	}
}
