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

// GetRoster returns one page of students paired with their progress using a
// constant number of queries (roster page + one batched progress lookup),
// regardless of roster size. The old shape issued a FindProgress query per
// student (N+1) — the batching lives here so the leak never crosses the
// repository seam.
func (r *moderatorRepo) GetRoster(ctx context.Context, page, limit int) ([]domain.RosterEntry, int64, int64, error) {
	var total int64
	var needsAttention int64

	if err := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ?", domain.RoleStudent).
		Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	if err := r.db.WithContext(ctx).
		Model(&domain.Progress{}).
		Joins("JOIN users ON users.id = progresses.learner_id AND users.role = ?", domain.RoleStudent).
		Where("progresses.current_streak = 0").
		Count(&needsAttention).Error; err != nil {
		return nil, 0, 0, err
	}

	offset := (page - 1) * limit
	var roster []domain.User
	if err := r.db.WithContext(ctx).
		Where("role = ?", domain.RoleStudent).
		Offset(offset).
		Limit(limit).
		Find(&roster).Error; err != nil {
		return nil, 0, 0, err
	}

	// Single batched progress lookup for the whole page (no per-learner loop).
	learnerIDs := make([]string, 0, len(roster))
	for _, u := range roster {
		learnerIDs = append(learnerIDs, u.ID)
	}
	progressByID := map[string]domain.Progress{}
	if len(learnerIDs) > 0 {
		var progresses []domain.Progress
		if err := r.db.WithContext(ctx).
			Where("learner_id IN ?", learnerIDs).
			Find(&progresses).Error; err != nil {
			return nil, 0, 0, err
		}
		for _, p := range progresses {
			progressByID[p.LearnerID] = p
		}
	}

	entries := make([]domain.RosterEntry, 0, len(roster))
	for _, u := range roster {
		entries = append(entries, domain.RosterEntry{User: u, Progress: progressByID[u.ID]})
	}
	return entries, total, needsAttention, nil
}
