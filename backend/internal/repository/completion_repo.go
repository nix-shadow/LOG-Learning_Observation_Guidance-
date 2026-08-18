package repository

import (
	"context"
	"fmt"
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
// Attempt semantics (derived from real client-reported facts):
//   - first completion        → record attempt (accuracy/elapsed), bump progress
//   - improving re-attempt    → keep best score, bump attempts, new guidance
//   - equal/lower replay       → refresh elapsed only; no progress/streak/guidance
//     changes (idempotent replay — offline queue flushes cannot double-bump)
func (r *completionRepo) CompleteActivityTx(ctx context.Context, learnerID, activityID string, stats domain.AttemptStats) (domain.Observation, domain.Guidance, error) {
	var obs domain.Observation
	var gui domain.Guidance

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var act domain.Activity
		if err := tx.First(&act, "id = ?", activityID).Error; err != nil {
			return fmt.Errorf("activity not found")
		}

		var la domain.LearnerActivity
		alreadyCompleted := true
		improved := false
		if err := tx.Where("learner_id = ? AND activity_id = ?", learnerID, activityID).First(&la).Error; err != nil {
			la = domain.LearnerActivity{
				LearnerID:  learnerID,
				ActivityID: activityID,
			}
			alreadyCompleted = false
		} else if la.Status != "Completed" {
			// Seeded / in-progress rows (e.g. demo student act-2) are the
			// learner's real first completion — count it like a new one.
			alreadyCompleted = false
		}

		if !alreadyCompleted {
			la.Status = "Completed"
			la.CompletedAt = time.Now()
			la.Score = stats.Score()
			la.Accuracy = stats.Accuracy()
			la.ElapsedSeconds = stats.ElapsedSeconds
			la.Attempts = 1
			if err := tx.Save(&la).Error; err != nil {
				return err
			}
		} else {
			// Replays and equal attempts only refresh the elapsed time — the
			// learner activity row is never downgraded and progress is never
			// double-bumped (offline queue flushes hit this path).
			improved = stats.HasQuiz() && stats.Accuracy() > la.Accuracy
			la.ElapsedSeconds = stats.ElapsedSeconds
			if improved {
				la.Score = stats.Score()
				la.Accuracy = stats.Accuracy()
				la.Attempts++
			}
			if err := tx.Save(&la).Error; err != nil {
				return err
			}
		}

		// Progress, streak, and chart rows are only touched on genuinely new
		// completions (an improving re-attempt already has them).
		if !alreadyCompleted {
			var progress domain.Progress
			if err := tx.Where("learner_id = ?", learnerID).First(&progress).Error; err != nil {
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

			// Date-aware streak: a same-day completion does not double the streak,
			// consecutive days increment it, and a gap resets it to 1.
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

			// Write a real DailyActivity row so chart-data reflects actual completions
			// (no fabricated fallback series).
			var daily domain.DailyActivity
			if err := tx.Where("learner_id = ? AND date = ?", learnerID, today).First(&daily).Error; err != nil {
				daily = domain.DailyActivity{
					ID:        service.GenerateSecureID("dly"),
					LearnerID: learnerID,
					Date:      today,
					DayName:   now.Weekday().String()[:3],
					Score:     100.0,
					Duration:  stats.ElapsedSeconds,
				}
				if err := tx.Create(&daily).Error; err != nil {
					return err
				}
			} else {
				daily.Score += 100.0
				daily.Duration += stats.ElapsedSeconds
				if err := tx.Save(&daily).Error; err != nil {
					return err
				}
			}
		}

		// Supportive observation + actionable guidance — text derived from real
		// accuracy when quiz data exists (see domain.AttemptStats).
		if !alreadyCompleted || improved {
			obs = domain.Observation{
				ID:        service.GenerateSecureID("obs"),
				LearnerID: learnerID,
				Category:  "strengths",
				Text:      stats.ObservationText(act.Title),
				CreatedAt: time.Now(),
			}
			if err := tx.Create(&obs).Error; err != nil {
				return err
			}

			gui = domain.Guidance{
				ID:        service.GenerateSecureID("gui"),
				LearnerID: learnerID,
				Text:      stats.GuidanceText(),
				Action:    "/learning",
				Type:      "next_step",
				CreatedAt: time.Now(),
			}
			return tx.Create(&gui).Error
		}
		return nil
	})

	if err != nil {
		return domain.Observation{}, domain.Guidance{}, err
	}
	return obs, gui, nil
}
