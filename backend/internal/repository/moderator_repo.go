package repository

import (
	"context"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type moderatorRepo struct {
	db *gorm.DB
}

func NewModeratorRepository(db *gorm.DB) domain.ModeratorRepository {
	return &moderatorRepo{db: db}
}

func (r *moderatorRepo) GetRoster(ctx context.Context, page, limit int) ([]domain.User, int64, int64, error) {
	var roster []domain.User
	var total int64
	var needsAttention int64

	r.db.WithContext(ctx).Model(&domain.User{}).Where("role = ?", domain.RoleStudent).Count(&total)

	r.db.WithContext(ctx).Model(&domain.Progress{}).
		Joins("JOIN users ON users.id = progresses.learner_id AND users.role = ?", domain.RoleStudent).
		Where("progresses.current_streak = 0").
		Count(&needsAttention)

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).
		Where("role = ?", domain.RoleStudent).
		Offset(offset).
		Limit(limit).
		Find(&roster).Error

	return roster, total, needsAttention, err
}
