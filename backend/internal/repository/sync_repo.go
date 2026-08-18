package repository

import (
	"context"
	"encoding/json"
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
			// offline quizzes land with real accuracy (parity with completion_repo).
			stats := domain.AttemptStats{}
			if req.Body != "" {
				if err := json.Unmarshal([]byte(req.Body), &stats); err != nil {
					failedCount++
					continue
				}
			}

			var learnerAct domain.LearnerActivity
			if err := tx.First(&learnerAct, "learner_id = ? AND activity_id = ?", learnerID, actID).Error; err != nil {
				learnerAct = domain.LearnerActivity{
					LearnerID:      learnerID,
					ActivityID:     actID,
					Status:         "Completed",
					CompletedAt:    time.Now(),
					Score:          stats.Score(),
					Accuracy:       stats.Accuracy(),
					ElapsedSeconds: stats.ElapsedSeconds,
					Attempts:       1,
				}
				if err := tx.Create(&learnerAct).Error; err != nil {
					return err
				}
			} else if learnerAct.Status == "Completed" {
				// Idempotent replay or improving re-attempt: never double-bump
				// progress/streak/score, but refresh elapsed and keep the best
				// score (mirrors completion_repo semantics).
				improved := stats.HasQuiz() && stats.Accuracy() > learnerAct.Accuracy
				learnerAct.ElapsedSeconds = stats.ElapsedSeconds
				if improved {
					learnerAct.Score = stats.Score()
					learnerAct.Accuracy = stats.Accuracy()
					learnerAct.Attempts++
				}
				if err := tx.Save(&learnerAct).Error; err != nil {
					return err
				}
				if !improved {
					// Replay: no new observation/guidance, no progress changes.
					continue
				}
			} else {
				learnerAct.Status = "Completed"
				learnerAct.CompletedAt = time.Now()
				learnerAct.Score = stats.Score()
				learnerAct.Accuracy = stats.Accuracy()
				learnerAct.ElapsedSeconds = stats.ElapsedSeconds
				learnerAct.Attempts = 1
				if err := tx.Save(&learnerAct).Error; err != nil {
					return err
				}
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

			// Date-aware streak (mirrors the online completion path):
			// same-day completions do not double the streak.
			now := time.Now()
			today := now.Truncate(24 * time.Hour)
			switch {
			case progress.LastActivityDate.IsZero():
				progress.CurrentStreak = 1
			case progress.LastActivityDate.Equal(today):
				// same-day replay: keep the streak as-is
			case progress.LastActivityDate.Equal(today.Add(-24 * time.Hour)):
				progress.CurrentStreak++
			default:
				progress.CurrentStreak = 1
			}
			progress.LastActivityDate = today

			if progress.OverallScore < 95.0 {
				progress.OverallScore += 2.5
			}
			if err := tx.Save(&progress).Error; err != nil {
				return err
			}

			// Real DailyActivity row so the chart-data endpoint reflects
			// syncs, not just online completions.
			var daily domain.DailyActivity
			if err := tx.Where("learner_id = ? AND date = ?", learnerID, today).First(&daily).Error; err != nil {
				daily = domain.DailyActivity{
					ID:        service.GenerateSecureID("dly"),
					LearnerID: learnerID,
					Date:      today,
					DayName:   now.Weekday().String()[:3],
					Score:     100.0,
					Duration:  0,
				}
				if err := tx.Create(&daily).Error; err != nil {
					return err
				}
			} else {
				daily.Score += 100.0
				if err := tx.Save(&daily).Error; err != nil {
					return err
				}
			}

			// Mirror the online completion flow: supportive observation
			// + actionable next-step guidance derived from real accuracy.
			if err := tx.Create(&domain.Observation{
				ID:        service.GenerateSecureID("obs"),
				LearnerID: learnerID,
				Category:  "strengths",
				Text:      stats.ObservationText(act.Title),
				CreatedAt: time.Now(),
			}).Error; err != nil {
				return err
			}
			if err := tx.Create(&domain.Guidance{
				ID:        service.GenerateSecureID("gui"),
				LearnerID: learnerID,
				Text:      stats.GuidanceText(),
				Action:    "/learning",
				Type:      "next_step",
				CreatedAt: time.Now(),
			}).Error; err != nil {
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
