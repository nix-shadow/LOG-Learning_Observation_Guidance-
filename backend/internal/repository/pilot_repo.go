package repository

import (
	"context"
	"time"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type pilotRepo struct {
	db *gorm.DB
}

func NewPilotRepository(db *gorm.DB) domain.PilotRepository {
	return &pilotRepo{db: db}
}

func (r *pilotRepo) CreateScan(ctx context.Context, scan *domain.PilotScan) error {
	return r.db.WithContext(ctx).Create(scan).Error
}

func (r *pilotRepo) MarkStarted(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&domain.PilotScan{}).
		Where("id = ?", id).Update("started", true).Error
}

func (r *pilotRepo) Stats(ctx context.Context) (domain.PilotStats, error) {
	var stats domain.PilotStats
	if err := r.db.WithContext(ctx).Model(&domain.PilotScan{}).Count(&stats.TotalScans).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.PilotScan{}).
		Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
		Count(&stats.ScansToday).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.PilotScan{}).
		Where("started = ?", true).Count(&stats.Starts).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.PilotScan{}).
		Distinct("poster_id").Count(&stats.DistinctPosters).Error; err != nil {
		return stats, err
	}
	// Start rate is derived from real rows: honest 0 when nothing started yet.
	if stats.TotalScans > 0 {
		stats.StartRate = float64(stats.Starts) / float64(stats.TotalScans)
	}
	if err := r.db.WithContext(ctx).Model(&domain.PilotScan{}).
		Select("poster_id, COUNT(*) as scans, SUM(CASE WHEN started = 1 THEN 1 ELSE 0 END) as starts").
		Group("poster_id").Order("scans desc").Scan(&stats.PerPoster).Error; err != nil {
		return stats, err
	}
	return stats, nil
}