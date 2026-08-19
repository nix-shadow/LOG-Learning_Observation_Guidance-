package repository

import (
	"context"
	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type activityRepo struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) domain.ActivityRepository {
	return &activityRepo{db: db}
}

func (r *activityRepo) FindAll(ctx context.Context) ([]domain.Activity, error) {
	var activities []domain.Activity
	if err := r.db.WithContext(ctx).Order("`order` asc").Find(&activities).Error; err != nil {
		return nil, err
	}
	return activities, nil
}
