package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"gorm.io/gorm"
)

type syncRepo struct {
	db *gorm.DB
}

func NewSyncRepository(db *gorm.DB) domain.SyncRepository {
	return &syncRepo{db: db}
}

func (r *syncRepo) SyncBulk(ctx context.Context, learnerID string, data []domain.SyncRequestItem) (int, error) {
	processedCount := 0

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, req := range data {
			if req.Method == "POST" && len(req.Endpoint) > 0 {
				// E.g. /activities/act-1/complete
				parts := strings.Split(req.Endpoint, "/")
				if len(parts) >= 4 && parts[1] == "activities" && parts[3] == "complete" {
					actID := parts[2]
					var act domain.Activity
					if err := tx.First(&act, "id = ?", actID).Error; err == nil {
						var learnerAct domain.LearnerActivity
						if err := tx.First(&learnerAct, "learner_id = ? AND activity_id = ?", learnerID, actID).Error; err != nil {
							learnerAct = domain.LearnerActivity{
								LearnerID:   learnerID,
								ActivityID:  actID,
								Status:      "Completed",
								CompletedAt: time.Now(),
								Score:       100.0,
							}
							tx.Create(&learnerAct)
						} else {
							learnerAct.Status = "Completed"
							learnerAct.CompletedAt = time.Now()
							tx.Save(&learnerAct)
						}

						// Scoped progress update: only touch the calling user's progress,
						// creating the record the first time a new learner syncs.
						var progress domain.Progress
						if err := tx.First(&progress, "learner_id = ?", learnerID).Error; err != nil {
							var totalTopics int64
							tx.Model(&domain.Activity{}).Count(&totalTopics)
							progress = domain.Progress{
								LearnerID:   learnerID,
								TotalTopics: int(totalTopics),
							}
						}
						progress.Completed++
						if progress.Completed > progress.TotalTopics {
							progress.Completed = progress.TotalTopics
						}
						progress.CurrentStreak++
						if progress.OverallScore < 95.0 {
							progress.OverallScore += 2.5
						}
						tx.Save(&progress)

						// Mirror the online completion flow: supportive observation
						// + actionable next-step guidance, generated after sync.
						obsTitle := "Module Completed"
						if act.Title != "" {
							obsTitle = act.Title
						}
						tx.Create(&domain.Observation{
							ID:        service.GenerateSecureID("obs"),
							LearnerID: learnerID,
							Category:  "strengths",
							Text:      fmt.Sprintf("Demonstrated excellent focus and successfully completed %s.", obsTitle),
							CreatedAt: time.Now(),
						})
						tx.Create(&domain.Guidance{
							ID:        service.GenerateSecureID("gui"),
							LearnerID: learnerID,
							Text:      "Great momentum! Continue to the next practice module to reinforce your logic skills.",
							Action:    "/learning",
							Type:      "next_step",
							CreatedAt: time.Now(),
						})
						processedCount++
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return 0, err
	}
	return processedCount, nil
}
