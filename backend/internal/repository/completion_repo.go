package repository

import (
	"context"
	"time"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"gorm.io/gorm"
)

type completionRepo struct {
	db *gorm.DB
}

func NewCompletionRepository(db *gorm.DB) domain.CompletionRepository {
	return &completionRepo{db: db}
}

// CompleteActivityTx runs the whole completion flow inside a single database
// transaction: learner activity upsert (with attempt facts), progress bump,
// supportive observation, and next-step guidance are all-or-nothing.
//
// The semantics live in the single shared seam applyCompletion
// (architecture review C2) — the online route and the offline flush
// (sync_repo.SyncBulk) can never drift. See completion_engine.go for the
// attempt/replay/needs-practice rules.
func (r *completionRepo) CompleteActivityTx(ctx context.Context, learnerID, activityID string, stats domain.AttemptStats) (domain.Observation, domain.Guidance, error) {
	var obs domain.Observation
	var gui domain.Guidance

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var act domain.Activity
		if err := tx.First(&act, "id = ?", activityID).Error; err != nil {
			return service.ErrActivityNotFound
		}
		var err error
		obs, gui, err = applyCompletion(tx, learnerID, activityID, act.Title, stats, time.Now())
		return err
	})

	if err != nil {
		return domain.Observation{}, domain.Guidance{}, err
	}
	return obs, gui, nil
}

// attemptAccuracy returns the completion accuracy on a 0-100 scale. Quiz-less
// attempts carry no accuracy signal, so they contribute 0 and are excluded by
// the weighted mean below — no invented numbers (AGENTS.md §1).
func attemptAccuracy(stats domain.AttemptStats) float64 {
	if !stats.HasQuiz() {
		return 0
	}
	return stats.Accuracy() * 100
}

// weightedMean folds a new sample into a running weighted mean.
func weightedMean(current float64, currentCount int, sample float64) float64 {
	return (current*float64(currentCount) + sample) / float64(currentCount+1)
}
