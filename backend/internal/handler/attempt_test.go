package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// setupAttemptRouter wires the learner completion endpoint with a stubbed
// auth middleware deriving the learner from a test header.
func setupAttemptRouter() (*gin.Engine, *LearnerHandler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", c.GetHeader("X-Test-Learner"))
		c.Next()
	})
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
		repository.NewCompletionRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(repository.NewModeratorRepository(database.DB))
	lh := NewLearnerHandler(learnerService, courseService, moderatorService)
	r.POST("/api/v1/activities/:id/complete", lh.CompleteActivity)
	return r, lh
}

func cleanupAttemptLearner(t *testing.T, learnerID string) {
	t.Helper()
	t.Cleanup(func() {
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Observation{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Guidance{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Progress{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.LearnerActivity{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.DailyActivity{})
		database.DB.Where("id = ?", learnerID).Delete(&domain.User{})
	})
}

func newAttemptLearner(t *testing.T) string {
	t.Helper()
	learnerID := service.GenerateSecureID("user")
	phone := service.GenerateSecureID("ph")
	database.DB.Create(&domain.User{ID: learnerID, Phone: &phone, Role: domain.RoleStudent, IsVerified: true})
	cleanupAttemptLearner(t, learnerID)
	return learnerID
}

func completeWithBody(t *testing.T, r *gin.Engine, learnerID, activityID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest("POST", "/api/v1/activities/"+activityID+"/complete", bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Test-Learner", learnerID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestCompleteActivityStoresAttemptAccuracy proves real quiz facts reach the
// server: accuracy, best score, elapsed seconds, and attempt count are
// persisted, and guidance derives from the actual accuracy band.
func TestCompleteActivityStoresAttemptAccuracy(t *testing.T) {
	learnerID := newAttemptLearner(t)
	r, _ := setupAttemptRouter()

	w := completeWithBody(t, r, learnerID, "act-1", `{"elapsed_seconds":120,"correct_count":3,"total_count":4}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %v: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"accuracy":0.75`) {
		t.Fatalf("expected accuracy 0.75 in response, got %s", w.Body.String())
	}

	var la domain.LearnerActivity
	if err := database.DB.First(&la, "learner_id = ? AND activity_id = ?", learnerID, "act-1").Error; err != nil {
		t.Fatalf("expected learner activity row: %v", err)
	}
	if la.Score != 75.0 {
		t.Errorf("expected best score 75, got %v", la.Score)
	}
	if la.Accuracy != 0.75 {
		t.Errorf("expected accuracy 0.75, got %v", la.Accuracy)
	}
	if la.ElapsedSeconds != 120 {
		t.Errorf("expected elapsed 120, got %v", la.ElapsedSeconds)
	}
	if la.Attempts != 1 {
		t.Errorf("expected attempts 1, got %v", la.Attempts)
	}

	// 0.75 lands in the "practice" band — supportive, never negative.
	var gui domain.Guidance
	if err := database.DB.Where("learner_id = ?", learnerID).Order("created_at desc").First(&gui).Error; err != nil {
		t.Fatalf("expected guidance row: %v", err)
	}
	if !strings.Contains(gui.Text, "practice") {
		t.Errorf("expected practice-band guidance, got %q", gui.Text)
	}
}

// TestReplayKeepsBestScore proves the approved attempt semantics: an equal
// replay refreshes elapsed time but never downgrades score or bumps attempts,
// while an improving attempt keeps the best score and records new guidance.
func TestReplayKeepsBestScore(t *testing.T) {
	learnerID := newAttemptLearner(t)
	r, _ := setupAttemptRouter()

	// First attempt: 2/4 = 50%.
	if w := completeWithBody(t, r, learnerID, "act-2", `{"elapsed_seconds":60,"correct_count":2,"total_count":4}`); w.Code != http.StatusOK {
		t.Fatalf("first attempt failed: %v %s", w.Code, w.Body.String())
	}

	// Equal replay (offline queue flush after a timeout): elapsed refreshes,
	// score and attempts stay put.
	if w := completeWithBody(t, r, learnerID, "act-2", `{"elapsed_seconds":90,"correct_count":2,"total_count":4}`); w.Code != http.StatusOK {
		t.Fatalf("replay failed: %v %s", w.Code, w.Body.String())
	}

	var la domain.LearnerActivity
	database.DB.First(&la, "learner_id = ? AND activity_id = ?", learnerID, "act-2")
	if la.Score != 50.0 {
		t.Errorf("replay must not downgrade score, got %v", la.Score)
	}
	if la.Attempts != 1 {
		t.Errorf("replay must not bump attempts, got %v", la.Attempts)
	}
	if la.ElapsedSeconds != 90 {
		t.Errorf("replay must refresh elapsed, got %v", la.ElapsedSeconds)
	}

	// Improving attempt: 4/4 = 100% — best score wins, attempts bump.
	if w := completeWithBody(t, r, learnerID, "act-2", `{"elapsed_seconds":45,"correct_count":4,"total_count":4}`); w.Code != http.StatusOK {
		t.Fatalf("improving attempt failed: %v %s", w.Code, w.Body.String())
	}
	database.DB.First(&la, "learner_id = ? AND activity_id = ?", learnerID, "act-2")
	if la.Score != 100.0 {
		t.Errorf("expected best score 100, got %v", la.Score)
	}
	if la.Attempts != 2 {
		t.Errorf("expected attempts 2 after improvement, got %v", la.Attempts)
	}

	var guiCount int64
	database.DB.Model(&domain.Guidance{}).Where("learner_id = ?", learnerID).Count(&guiCount)
	if guiCount != 2 {
		t.Errorf("expected 2 guidance rows (first + improving), got %v", guiCount)
	}
}

// TestGuidanceThresholds proves each accuracy band produces its own
// supportive guidance copy (≥80% strengths, 50-79% practice, <50% foundations).
func TestGuidanceThresholds(t *testing.T) {
	r, _ := setupAttemptRouter()

	cases := []struct {
		accuracyBody string
		wantSubstr   string
	}{
		{`{"correct_count":4,"total_count":4}`, "Keep up the great work"},
		{`{"correct_count":3,"total_count":4}`, "could use more practice"},
		{`{"correct_count":1,"total_count":4}`, "strengthen the foundations"},
	}

	for _, tc := range cases {
		learnerID := newAttemptLearner(t)
		if w := completeWithBody(t, r, learnerID, "act-1", tc.accuracyBody); w.Code != http.StatusOK {
			t.Fatalf("completion failed: %v %s", w.Code, w.Body.String())
		}
		var gui domain.Guidance
		if err := database.DB.Where("learner_id = ?", learnerID).Order("created_at desc").First(&gui).Error; err != nil {
			t.Fatalf("expected guidance: %v", err)
		}
		if !strings.Contains(gui.Text, tc.wantSubstr) {
			t.Errorf("expected guidance containing %q, got %q", tc.wantSubstr, gui.Text)
		}
	}
}

// TestCompleteActivityTransitionsInProgressRow proves a pre-existing
// "In progress" row (e.g. the seeded demo student) is a real first completion:
// it flips to Completed, records the attempt facts, and bumps progress — while
// an already-Completed row still behaves as an idempotent replay.
func TestCompleteActivityTransitionsInProgressRow(t *testing.T) {
	learnerID := newAttemptLearner(t)
	database.DB.Create(&domain.LearnerActivity{
		LearnerID:  learnerID,
		ActivityID: "act-2",
		Status:     "In progress",
		Score:      50.0,
	})

	r, _ := setupAttemptRouter()
	if w := completeWithBody(t, r, learnerID, "act-2", `{"elapsed_seconds":60,"correct_count":4,"total_count":4}`); w.Code != http.StatusOK {
		t.Fatalf("completion failed: %v %s", w.Code, w.Body.String())
	}

	var la domain.LearnerActivity
	database.DB.First(&la, "learner_id = ? AND activity_id = ?", learnerID, "act-2")
	if la.Status != "Completed" {
		t.Errorf("expected status Completed, got %q", la.Status)
	}
	if la.Score != 100.0 {
		t.Errorf("expected score 100, got %v", la.Score)
	}
	if la.Attempts != 1 {
		t.Errorf("expected attempts 1, got %v", la.Attempts)
	}

	var progress domain.Progress
	database.DB.First(&progress, "learner_id = ?", learnerID)
	if progress.Completed != 1 {
		t.Errorf("expected progress completed 1, got %v", progress.Completed)
	}

	// A second completion of the now-Completed row must not double-bump.
	if w := completeWithBody(t, r, learnerID, "act-2", `{"elapsed_seconds":30,"correct_count":4,"total_count":4}`); w.Code != http.StatusOK {
		t.Fatalf("replay failed: %v %s", w.Code, w.Body.String())
	}
	database.DB.First(&progress, "learner_id = ?", learnerID)
	if progress.Completed != 1 {
		t.Errorf("replay must not double-bump progress, got %v", progress.Completed)
	}
	var guiCount int64
	database.DB.Model(&domain.Guidance{}).Where("learner_id = ?", learnerID).Count(&guiCount)
	if guiCount != 1 {
		t.Errorf("replay must not create new guidance, got %v", guiCount)
	}
}

// TestCompleteActivityWithoutQuizData proves legacy clients (empty body) keep
// the encouragement flow and are never scored on fabricated data.
func TestCompleteActivityWithoutQuizData(t *testing.T) {
	learnerID := newAttemptLearner(t)
	r, _ := setupAttemptRouter()

	if w := completeWithBody(t, r, learnerID, "act-1", ""); w.Code != http.StatusOK {
		t.Fatalf("expected 200 for legacy completion, got %v: %s", w.Code, w.Body.String())
	}
	var la domain.LearnerActivity
	if err := database.DB.First(&la, "learner_id = ? AND activity_id = ?", learnerID, "act-1").Error; err != nil {
		t.Fatalf("expected learner activity: %v", err)
	}
	if la.Accuracy != 0 {
		t.Errorf("no quiz data must yield zero accuracy, got %v", la.Accuracy)
	}
	var gui domain.Guidance
	database.DB.Where("learner_id = ?", learnerID).Order("created_at desc").First(&gui)
	if !strings.Contains(gui.Text, "Great momentum") {
		t.Errorf("expected legacy encouragement, got %q", gui.Text)
	}
}

// TestSyncBulkAttemptParity proves the offline path parses the same attempt
// payload and lands the same accuracy fields as the online completion.
func TestSyncBulkAttemptParity(t *testing.T) {
	learnerID := newAttemptLearner(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", learnerID)
		c.Next()
	})
	syncHandler := NewSyncHandler(service.NewSyncService(repository.NewSyncRepository(database.DB)))
	r.POST("/api/v1/sync/bulk", syncHandler.SyncBulk)

	payload := `{"version":"1.0","data":[{"endpoint":"/activities/act-1/complete","method":"POST","body":"{\"elapsed_seconds\":200,\"correct_count\":4,\"total_count\":4}"}]}`
	req, _ := http.NewRequest("POST", "/api/v1/sync/bulk", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("sync failed: %v %s", w.Code, w.Body.String())
	}

	var la domain.LearnerActivity
	if err := database.DB.First(&la, "learner_id = ? AND activity_id = ?", learnerID, "act-1").Error; err != nil {
		t.Fatalf("expected learner activity from sync: %v", err)
	}
	if la.Score != 100.0 {
		t.Errorf("expected score 100 from sync payload, got %v", la.Score)
	}
	if la.ElapsedSeconds != 200 {
		t.Errorf("expected elapsed 200 from sync payload, got %v", la.ElapsedSeconds)
	}

	var gui domain.Guidance
	if err := database.DB.Where("learner_id = ?", learnerID).Order("created_at desc").First(&gui).Error; err != nil {
		t.Fatalf("expected guidance from sync: %v", err)
	}
	if !strings.Contains(gui.Text, "Keep up the great work") {
		t.Errorf("expected high-accuracy guidance from sync, got %q", gui.Text)
	}
}
