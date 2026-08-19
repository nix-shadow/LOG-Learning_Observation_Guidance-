package repository

import (
	"context"
	"log-backend/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type progressRepo struct {
	db *gorm.DB
}

func NewProgressRepository(db *gorm.DB) domain.ProgressRepository {
	return &progressRepo{db: db}
}

func (r *progressRepo) FindLearnerActivities(ctx context.Context, learnerID string) ([]domain.LearnerActivity, error) {
	var activities []domain.LearnerActivity
	if err := r.db.WithContext(ctx).Where("learner_id = ?", learnerID).Find(&activities).Error; err != nil {
		return nil, err
	}
	return activities, nil
}

func (r *progressRepo) FindProgress(ctx context.Context, learnerID string) (*domain.Progress, error) {
	var progress domain.Progress
	if err := r.db.WithContext(ctx).Where("learner_id = ?", learnerID).First(&progress).Error; err != nil {
		return nil, err
	}
	return &progress, nil
}

func (r *progressRepo) SaveProgress(ctx context.Context, progress *domain.Progress) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "learner_id"}},
		UpdateAll: true,
	}).Create(progress).Error
}
