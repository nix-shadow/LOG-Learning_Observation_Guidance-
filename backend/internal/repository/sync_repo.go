package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type syncRepo struct {
	db *gorm.DB
}

func NewSyncRepository(db *gorm.DB) domain.SyncRepository {
	return &syncRepo{db: db}
}

// SyncBulk replays a batch of offline-queued requests inside one transaction.
// Only completion-shaped items are accepted; anything else is counted as
// failed. Each item delegates to the single shared completion seam
// applyCompletion (architecture review C2) — the offline flush and the
// online route (completion_repo.CompleteActivityTx) share one implementation
// and can never drift.
func (r *syncRepo) SyncBulk(ctx context.Context, learnerID string, data []domain.SyncRequestItem) (int, int, error) {
	processedCount := 0
	failedCount := 0

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, req := range data {
			if req.Method != "POST" || len(req.Endpoint) == 0 {
				failedCount++
				continue
			}
			// E.g. /activities/act-1/complete
			parts := strings.Split(req.Endpoint, "/")
			if len(parts) < 4 || parts[1] != "activities" || parts[3] != "complete" {
				failedCount++
				continue
			}
			actID := parts[2]
			var act domain.Activity
			if err := tx.First(&act, "id = ?", actID).Error; err != nil {
				// Unknown activity: report honestly instead of masking the failure.
				failedCount++
				continue
			}

			// Parse the same attempt payload the online completion accepts so
			// offline quizzes land with real accuracy (parity with the seam).
			stats := domain.AttemptStats{}
			if req.Body != "" {
				if err := json.Unmarshal([]byte(req.Body), &stats); err != nil {
					failedCount++
					continue
				}
			}
			stats = stats.Clamp()

			if _, _, err := applyCompletion(tx, learnerID, actID, act.Title, stats, time.Now()); err != nil {
				// A write failure aborts the whole flush — partial syncs must
				// never be reported as complete (the queue retries the rest).
				return err
			}
			processedCount++
		}
		return nil
	})

	if err != nil {
		return 0, 0, err
	}
	return processedCount, failedCount, nil
}
