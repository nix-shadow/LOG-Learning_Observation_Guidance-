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

func (r *activityRepo) FindByID(ctx context.Context, id string) (*domain.Activity, error) {
	var act domain.Activity
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&act).Error; err != nil {
		return nil, err
	}
	return &act, nil
}

// CreateMany inserts a batch of activities in one transaction (WP-3.1 import
// pipeline). Existing IDs are skipped so learner progress against an activity
// is never orphaned by a re-import.
func (r *activityRepo) CreateMany(ctx context.Context, acts []domain.Activity) (imported int, skipped int, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range acts {
			var existing int64
			if err := tx.Model(&domain.Activity{}).Where("id = ?", acts[i].ID).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				skipped++
				continue
			}
			if err := tx.Create(&acts[i]).Error; err != nil {
				return err
			}
			imported++
		}
		return nil
	})
	return imported, skipped, err
}
