package repository

import (
	"context"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	Repository[model.User]
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		Repository: Repository[model.User]{db},
	}
}

// CountByEmail returns the number of users with the given email.
func (r *UserRepository) CountByEmail(ctx context.Context, email string) (int64, error) {
	var total int64
	var user model.User
	err := r.Repository.getDb(ctx).Model(&user).Where("email = ?", email).Count(&total).Error
	return total, err
}

// FindByEmail finds a user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, user *model.User, email string) error {
	return r.Repository.getDb(ctx).Where("email = ?", email).First(&user).Error
}

// FindByUUID finds a user by UUID, with roles and permissions preloaded.
func (r *UserRepository) FindByUUID(ctx context.Context, user *model.User, uuid string) error {
	return r.Repository.getDb(ctx).Preload("Roles").
		Preload("Permissions").
		Preload("Roles.Permissions").
		Where("uuid = ?", uuid).
		First(&user).Error
}

// Search returns a list of users and total count based on filter and pagination.
func (r *UserRepository) Search(ctx context.Context, request *dto.SearchUserRequest) ([]*model.User, int64, error) {
	db := r.Repository.getDb(ctx).Scopes(r.FilterUser(request))

	var user []*model.User
	if err := db.Offset((request.Page - 1) * request.Size).Limit(request.Size).Find(&user).Error; err != nil {
		return nil, 0, err
	}

	var total int64 = 0
	if err := db.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return user, total, nil
}

// FilterUser returns a GORM scope for filtering users by name or email.
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
