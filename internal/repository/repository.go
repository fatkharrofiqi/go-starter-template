package repository

import (
	"context"

	"gorm.io/gorm"
)

type contextKey string

var TxKey contextKey = "tx"

type Repository[T any] struct {
	DB *gorm.DB
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	return r.getDb(ctx).Create(entity).Error
}

func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	return r.getDb(ctx).Save(entity).Error
}

func (r *Repository[T]) Delete(ctx context.Context, entity *T) error {
	return r.getDb(ctx).Delete(entity).Error
}

func (r *Repository[T]) CountById(ctx context.Context, id any) (int64, error) {
	var total int64
	err := r.getDb(ctx).Model(new(T)).Where("id = ?", id).Count(&total).Error
	return total, err
}

func (r *Repository[T]) FindById(ctx context.Context, entity *T, id any) error {
	return r.getDb(ctx).Where("id = ?", id).Take(entity).Error
}

func (r *Repository[T]) getDb(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(TxKey).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}
