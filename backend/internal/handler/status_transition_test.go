package handler

import (
	"context"
	"testing"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"
)

// WP-1.1 RC-01 status-transition tests. They exercise the real completion
// transaction against seeded activities and assert the canonical per-learner
// status vocabulary: not-started → active → completed / needs-practice, with
// idempotent replays never double-bumping progress.

func completeAttempt(t *testing.T, learnerID, activityID string, correct, total int) (domain.Observation, domain.Guidance) {
	t.Helper()
	repo := repository.NewCompletionRepository(database.DB)
	obs, gui, err := repo.CompleteActivityTx(context.Background(), learnerID, activityID, domain.AttemptStats{
		ElapsedSeconds: 120,
		CorrectCount:   correct,
		TotalCount:     total,
	})
	if err != nil {
		t.Fatalf("completion failed: %v", err)
	}
	return obs, gui
}

func storedActivity(t *testing.T, learnerID, activityID string) domain.LearnerActivity {
	t.Helper()
	var la domain.LearnerActivity
	if err := database.DB.Where("learner_id = ? AND activity_id = ?", learnerID, activityID).First(&la).Error; err != nil {
		t.Fatalf("learner activity row missing: %v", err)
	}
	return la
}

func TestStatusTransitionHighAccuracyCompletes(t *testing.T) {
	learnerID := newAttemptLearner(t)

	completeAttempt(t, learnerID, "act-1", 10, 10)

	la := storedActivity(t, learnerID, "act-1")
	if la.Status != domain.StatusCompleted {
		t.Fatalf("expected %q after a perfect attempt, got %q", domain.StatusCompleted, la.Status)
	}
	if la.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", la.Attempts)
	}

	var progress domain.Progress
	if err := database.DB.Where("learner_id = ?", learnerID).First(&progress).Error; err != nil {
		t.Fatalf("progress row missing: %v", err)
	}
	if progress.Completed != 1 {
		t.Fatalf("expected progress.Completed=1, got %d", progress.Completed)
	}
}

func TestStatusTransitionLowAccuracyFlagsNeedsPractice(t *testing.T) {
	learnerID := newAttemptLearner(t)

	completeAttempt(t, learnerID, "act-1", 4, 10)

	la := storedActivity(t, learnerID, "act-1")
	if la.Status != domain.StatusNeedsPractice {
		t.Fatalf("expected %q for 40%% accuracy, got %q", domain.StatusNeedsPractice, la.Status)
	}
	if la.Status == "failed" || la.Status == "Failed" {
		t.Fatalf("status vocabulary must never contain 'failed': got %q", la.Status)
	}

	// The needs-practice completion still counts as a real completion.
	var progress domain.Progress
	if err := database.DB.Where("learner_id = ?", learnerID).First(&progress).Error; err != nil {
		t.Fatalf("progress row missing: %v", err)
	}
	if progress.Completed != 1 {
		t.Fatalf("expected progress.Completed=1, got %d", progress.Completed)
	}
}

func TestStatusTransitionImprovingReattemptClearsFlag(t *testing.T) {
	learnerID := newAttemptLearner(t)

	completeAttempt(t, learnerID, "act-1", 3, 10) // 30% → needs-practice
	if la := storedActivity(t, learnerID, "act-1"); la.Status != domain.StatusNeedsPractice {
		t.Fatalf("expected needs-practice after 30%%, got %q", la.Status)
	}

	completeAttempt(t, learnerID, "act-1", 9, 10) // 90% → clears the flag

	la := storedActivity(t, learnerID, "act-1")
	if la.Status != domain.StatusCompleted {
		t.Fatalf("expected %q after improving re-attempt, got %q", domain.StatusCompleted, la.Status)
	}
	if la.Attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", la.Attempts)
	}

	// The re-attempt must NOT double-bump progress.
	var progress domain.Progress
	if err := database.DB.Where("learner_id = ?", learnerID).First(&progress).Error; err != nil {
		t.Fatalf("progress row missing: %v", err)
	}
	if progress.Completed != 1 {
		t.Fatalf("expected progress.Completed=1 (no double-bump), got %d", progress.Completed)
	}
}

func TestStatusTransitionImprovingReattemptBelowThresholdKeepsFlag(t *testing.T) {
	learnerID := newAttemptLearner(t)

	completeAttempt(t, learnerID, "act-1", 2, 10) // 20% → needs-practice
	completeAttempt(t, learnerID, "act-1", 5, 10) // 50% → improved but still below threshold

	la := storedActivity(t, learnerID, "act-1")
	if la.Status != domain.StatusNeedsPractice {
		t.Fatalf("expected needs-practice to persist below the threshold, got %q", la.Status)
	}
	if la.Attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", la.Attempts)
	}
}

func TestStatusTransitionIdempotentReplay(t *testing.T) {
	learnerID := newAttemptLearner(t)

	completeAttempt(t, learnerID, "act-1", 8, 10)
	first := storedActivity(t, learnerID, "act-1")

	// Equal replay (offline queue flush hitting the same payload twice).
	completeAttempt(t, learnerID, "act-1", 8, 10)
	second := storedActivity(t, learnerID, "act-1")

	if second.Status != first.Status {
		t.Fatalf("replay changed status: %q → %q", first.Status, second.Status)
	}
	if second.Attempts != first.Attempts {
		t.Fatalf("replay changed attempts: %d → %d", first.Attempts, second.Attempts)
	}

	var progress domain.Progress
	if err := database.DB.Where("learner_id = ?", learnerID).First(&progress).Error; err != nil {
		t.Fatalf("progress row missing: %v", err)
	}
	if progress.Completed != 1 {
		t.Fatalf("replay double-bumped progress: got %d", progress.Completed)
	}
}

func TestStatusTransitionNoQuizCompletes(t *testing.T) {
	learnerID := newAttemptLearner(t)

	// A completion without knowledge checks has no accuracy signal to judge —
	// it must be "completed", never "needs-practice".
	repo := repository.NewCompletionRepository(database.DB)
	if _, _, err := repo.CompleteActivityTx(context.Background(), learnerID, "act-1", domain.AttemptStats{
		ElapsedSeconds: 60,
	}); err != nil {
		t.Fatalf("completion failed: %v", err)
	}

	if la := storedActivity(t, learnerID, "act-1"); la.Status != domain.StatusCompleted {
		t.Fatalf("expected %q for a quiz-less completion, got %q", domain.StatusCompleted, la.Status)
	}
}

func TestDashboardExposesCanonicalStatuses(t *testing.T) {
	learnerID := newAttemptLearner(t)

	completeAttempt(t, learnerID, "act-1", 4, 10) // needs-practice

	svc := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
		repository.NewCompletionRepository(database.DB),
	)
	_, _, activities, _, _, err := svc.GetDashboardData(context.Background(), learnerID)
	if err != nil {
		t.Fatalf("dashboard failed: %v", err)
	}

	statusByID := map[string]string{}
	for _, a := range activities {
		statusByID[a.ID] = a.Status
	}
	if statusByID["act-1"] != domain.StatusNeedsPractice {
		t.Fatalf("expected act-1 status %q from the API, got %q", domain.StatusNeedsPractice, statusByID["act-1"])
	}
	for id, status := range statusByID {
		switch status {
		case domain.StatusNotStarted, domain.StatusActive, domain.StatusNeedsPractice, domain.StatusCompleted:
		default:
			t.Fatalf("activity %s leaked a non-canonical status: %q", id, status)
		}
	}
}

func TestResolveActivityStatusLegacyRows(t *testing.T) {
	cases := []struct {
		name string
		la   domain.LearnerActivity
		want string
	}{
		{"empty row", domain.LearnerActivity{}, domain.StatusNotStarted},
		{"pending legacy", domain.LearnerActivity{Status: "Pending"}, domain.StatusNotStarted},
		{"in-progress legacy", domain.LearnerActivity{Status: "In Progress"}, domain.StatusActive},
		{"in-progress legacy seed casing", domain.LearnerActivity{Status: "In progress"}, domain.StatusActive},
		{"active canonical", domain.LearnerActivity{Status: domain.StatusActive}, domain.StatusActive},
		{"completed canonical", domain.LearnerActivity{Status: domain.StatusCompleted}, domain.StatusCompleted},
		{"needs-practice canonical", domain.LearnerActivity{Status: domain.StatusNeedsPractice}, domain.StatusNeedsPractice},
		{"completed legacy with low accuracy", domain.LearnerActivity{Status: "Completed", Accuracy: 0.4}, domain.StatusNeedsPractice},
		{"completed legacy with no accuracy signal", domain.LearnerActivity{Status: "Completed", Accuracy: 0}, domain.StatusCompleted},
		{"attempts without completion", domain.LearnerActivity{Attempts: 2}, domain.StatusActive},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.ResolveActivityStatus(tc.la); got != tc.want {
				t.Fatalf("ResolveActivityStatus(%+v) = %q, want %q", tc.la, got, tc.want)
			}
		})
	}
}
