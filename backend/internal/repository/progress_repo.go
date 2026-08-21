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

// FindLearnerActivitiesBatch (WP-2.3): one query for the whole class, keyed
// by learner ID — the gradebook never loops per student.
func (r *progressRepo) FindLearnerActivitiesBatch(ctx context.Context, learnerIDs []string) (map[string][]domain.LearnerActivity, error) {
	out := map[string][]domain.LearnerActivity{}
	if len(learnerIDs) == 0 {
		return out, nil
	}
	var activities []domain.LearnerActivity
	if err := r.db.WithContext(ctx).
		Where("learner_id IN ?", learnerIDs).
		Find(&activities).Error; err != nil {
		return nil, err
	}
	for _, a := range activities {
		out[a.LearnerID] = append(out[a.LearnerID], a)
	}
	return out, nil
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
