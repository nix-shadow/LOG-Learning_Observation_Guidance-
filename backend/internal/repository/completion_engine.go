package repository

import (
	"time"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"gorm.io/gorm"
)

// applyCompletion runs the entire best-score completion flow inside the
// given transaction. It is the ONE seam where completion semantics live
// (architecture review C2): the online route (CompleteActivityTx) and the
// offline flush (SyncBulk) both delegate here, so the two paths can never
// drift.
//
// Attempt semantics (derived from real client-reported facts):
//   - first completion        → record attempt (accuracy/elapsed), bump progress
//   - improving re-attempt    → keep best score, bump attempts, new guidance
//   - equal/lower replay       → refresh elapsed only; no progress/streak/guidance
//     changes (idempotent replay — offline queue flushes cannot double-bump)
//
// Needs-practice determination (WP-1.1 RC-01): a completion with quiz data
// below NeedsPracticeAccuracyThreshold is flagged "needs-practice" instead
// of "completed" — supportive framing, never "failed". An improving
// re-attempt that crosses the threshold clears the flag back to
// "completed". Completions without quiz data are always "completed" (there
// is no accuracy signal to judge).
//
// activityTitle feeds the supportive observation text; the caller is
// responsible for loading the activity (the online route reports 404, the
// offline flush counts a failed item — different error policies, one seam).
func applyCompletion(tx *gorm.DB, learnerID, activityID, activityTitle string, stats domain.AttemptStats, now time.Time) (domain.Observation, domain.Guidance, error) {
	var obs domain.Observation
	var gui domain.Guidance

	var la domain.LearnerActivity
	alreadyCompleted := true
	improved := false
	if err := tx.Where("learner_id = ? AND activity_id = ?", learnerID, activityID).First(&la).Error; err != nil {
		la = domain.LearnerActivity{
			LearnerID:  learnerID,
			ActivityID: activityID,
		}
		alreadyCompleted = false
	} else if la.Status != domain.StatusCompleted && la.Status != domain.StatusNeedsPractice {
		// Seeded / in-progress rows (e.g. demo student act-2) are the
		// learner's real first completion — count it like a new one.
		alreadyCompleted = false
	}

	if !alreadyCompleted {
		la.Status = domain.StatusCompleted
		if stats.HasQuiz() && stats.Accuracy() < domain.NeedsPracticeAccuracyThreshold {
			la.Status = domain.StatusNeedsPractice
		}
		// WP-0.2 research round: record the client's real completion
		// instant (clamped) instead of server receive time, so offline
		// flushes date the work when it happened, not when it synced.
		la.CompletedAt = stats.CompletedAt(now)
		la.Score = stats.Score()
		la.Accuracy = stats.Accuracy()
		la.ElapsedSeconds = stats.ElapsedSeconds
		la.Attempts = 1
		if err := tx.Save(&la).Error; err != nil {
			return obs, gui, err
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
			// Crossing the threshold clears the needs-practice flag;
			// staying below it keeps the flag (supportive, not punitive).
			if la.Accuracy >= domain.NeedsPracticeAccuracyThreshold {
				la.Status = domain.StatusCompleted
			}
		}
		if err := tx.Save(&la).Error; err != nil {
			return obs, gui, err
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
		// consecutive days increment it, and a gap resets it to 1. The
		// calendar date is the learner's local date (their timezone), so a
		// completion at 23:00 Kathmandu time counts on the right day.
		completedAt := stats.CompletedAt(now)
		loc := stats.Location()
		local := completedAt.In(loc)
		today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
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
			return obs, gui, err
		}

		// Write a real DailyActivity row so chart-data reflects actual completions
		// (no fabricated fallback series). Date + day name come from the
		// learner's local calendar, not server UTC.
		var daily domain.DailyActivity
		if err := tx.Where("learner_id = ? AND date = ?", learnerID, today).First(&daily).Error; err != nil {
			daily = domain.DailyActivity{
				ID:        service.GenerateSecureID("dly"),
				LearnerID: learnerID,
				Date:      today,
				DayName:   local.Weekday().String()[:3],
				Score:     100.0,
				Duration:  stats.ElapsedSeconds,
				Attempts:  1,
				Accuracy:  attemptAccuracy(stats),
			}
			if err := tx.Create(&daily).Error; err != nil {
				return obs, gui, err
			}
		} else {
			daily.Score += 100.0
			daily.Duration += stats.ElapsedSeconds
			// WP-1.2 RC-02 practice metrics: running weighted mean.
			daily.Accuracy = weightedMean(daily.Accuracy, daily.Attempts, attemptAccuracy(stats))
			daily.Attempts++
			if err := tx.Save(&daily).Error; err != nil {
				return obs, gui, err
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
			Text:      stats.ObservationText(activityTitle),
			CreatedAt: now,
		}
		if err := tx.Create(&obs).Error; err != nil {
			return obs, gui, err
		}

		gui = domain.Guidance{
			ID:        service.GenerateSecureID("gui"),
			LearnerID: learnerID,
			Text:      stats.GuidanceText(),
			Action:    "/learning",
			Type:      "next_step",
			CreatedAt: now,
		}
		return obs, gui, tx.Create(&gui).Error
	}
	return obs, gui, nil
}
