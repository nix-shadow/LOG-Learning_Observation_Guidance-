package repository

import (
	"context"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type learnerDataRepo struct {
	db *gorm.DB
}

func NewLearnerDataRepository(db *gorm.DB) domain.LearnerDataRepository {
	return &learnerDataRepo{db: db}
}

func (r *learnerDataRepo) FindObservations(ctx context.Context, learnerID string) ([]domain.Observation, error) {
	var obs []domain.Observation
	err := r.db.WithContext(ctx).Order("created_at desc").Where("learner_id = ?", learnerID).Find(&obs).Error
	return obs, err
}

func (r *learnerDataRepo) FindGuidance(ctx context.Context, learnerID string) ([]domain.Guidance, error) {
	var gui []domain.Guidance
	err := r.db.WithContext(ctx).Order("created_at desc").Where("learner_id = ?", learnerID).Find(&gui).Error
	return gui, err
}

func (r *learnerDataRepo) FindDailyActivities(ctx context.Context, learnerID string) ([]domain.DailyActivity, error) {
	var acts []domain.DailyActivity
	err := r.db.WithContext(ctx).Where("learner_id = ?", learnerID).Order("date asc").Find(&acts).Error
	return acts, err
}

func (r *learnerDataRepo) SaveObservation(ctx context.Context, obs *domain.Observation) error {
	return r.db.WithContext(ctx).Create(obs).Error
}

func (r *learnerDataRepo) SaveGuidance(ctx context.Context, gui *domain.Guidance) error {
	return r.db.WithContext(ctx).Create(gui).Error
}
