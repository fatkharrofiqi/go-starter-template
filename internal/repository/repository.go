package repository

import (
	"context"

	"gorm.io/gorm"
)

type Repository[T any] struct {
	DB *gorm.DB
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Create(entity).Error
}

func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Save(entity).Error
}

func (r *Repository[T]) Delete(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Delete(entity).Error
}

func (r *Repository[T]) CountById(ctx context.Context, id any) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Model(new(T)).Where("id = ?", id).Count(&total).Error
	return total, err
}

func (r *Repository[T]) FindById(ctx context.Context, entity *T, id any) error {
	return r.DB.WithContext(ctx).Where("id = ?", id).Take(entity).Error
}
