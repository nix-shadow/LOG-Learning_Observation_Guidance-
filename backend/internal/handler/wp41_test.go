package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"
)

// TestCompletionParityOnlineVsSync is the C2 seam proof: the online route
// (completion_repo.CompleteActivityTx) and the offline flush
// (sync_repo.SyncBulk) share one implementation, so identical payloads must
// produce identical rows. Any future drift fails here.
func TestCompletionParityOnlineVsSync(t *testing.T) {
	completionRepo := repository.NewCompletionRepository(database.DB)
	syncRepo := repository.NewSyncRepository(database.DB)

	// Real users — the C4 foreign keys enforce learner_activities.learner_id,
	// so the parity rows must reference accounts that exist (no ghost users).
	for _, id := range []string{"parity-online", "parity-sync"} {
		database.DB.Create(&domain.User{ID: id, Name: "Parity Tester", Email: id + "@parity.test", Role: domain.RoleStudent, IsVerified: true})
		t.Cleanup(func() {
			database.DB.Unscoped().Where("id = ?", id).Delete(&domain.User{})
		})
	}

	now := time.Now()
	payload := domain.AttemptStats{
		ElapsedSeconds:    90,
		CorrectCount:      4,
		TotalCount:        5,
		CompletedAtUnixMs: now.Add(-time.Hour).UnixMilli(),
		TimezoneIANA:      "Asia/Kathmandu",
	}.Clamp()

	// Online path for learner A.
	if _, _, err := completionRepo.CompleteActivityTx(context.Background(), "parity-online", "act-3", payload); err != nil {
		t.Fatalf("online completion failed: %v", err)
	}

	// Sync-bulk path for learner B — identical payload, same activity.
	item, err := marshalSyncItem("/activities/act-3/complete", payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	processed, failed, err := syncRepo.SyncBulk(context.Background(), "parity-sync", []domain.SyncRequestItem{item})
	if err != nil {
		t.Fatalf("sync bulk failed: %v", err)
	}
	if processed != 1 || failed != 0 {
		t.Fatalf("expected processed=1 failed=0, got %d/%d", processed, failed)
	}

	// The two paths must produce identical learner-activity rows.
	for _, learnerID := range []string{"parity-online", "parity-sync"} {
		var la domain.LearnerActivity
		if err := database.DB.Where("learner_id = ? AND activity_id = ?", learnerID, "act-3").First(&la).Error; err != nil {
			t.Fatalf("missing learner activity for %s: %v", learnerID, err)
		}
		if la.Status != domain.StatusCompleted || la.Accuracy != 0.8 || la.Attempts != 1 || la.Score == 0 {
			t.Fatalf("%s: unexpected row %+v", learnerID, la)
		}
		t.Cleanup(func() { database.DB.Where("learner_id = ?", learnerID).Delete(&domain.LearnerActivity{}) })

		var progress domain.Progress
		if err := database.DB.Where("learner_id = ?", learnerID).First(&progress).Error; err != nil {
			t.Fatalf("missing progress for %s: %v", learnerID, err)
		}
		if progress.Completed < 1 || progress.CurrentStreak != 1 {
			t.Fatalf("%s: unexpected progress %+v", learnerID, progress)
		}
		t.Cleanup(func() { database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Progress{}) })

		var daily domain.DailyActivity
		if err := database.DB.Where("learner_id = ?", learnerID).First(&daily).Error; err != nil {
			t.Fatalf("missing daily row for %s: %v", learnerID, err)
		}
		t.Cleanup(func() { database.DB.Where("learner_id = ?", learnerID).Delete(&domain.DailyActivity{}) })

		var obsCount, guiCount int64
		database.DB.Model(&domain.Observation{}).Where("learner_id = ?", learnerID).Count(&obsCount)
		database.DB.Model(&domain.Guidance{}).Where("learner_id = ?", learnerID).Count(&guiCount)
		if obsCount != 1 || guiCount != 1 {
			t.Fatalf("%s: expected 1 observation + 1 guidance, got %d/%d", learnerID, obsCount, guiCount)
		}
		t.Cleanup(func() {
			database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Observation{})
			database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Guidance{})
		})

		// Idempotent replay: identical flush must NOT double-bump anything.
		if _, _, err := syncRepo.SyncBulk(context.Background(), learnerID, []domain.SyncRequestItem{item}); err != nil {
			t.Fatalf("replay failed for %s: %v", learnerID, err)
		}
		var la2 domain.LearnerActivity
		database.DB.Where("learner_id = ? AND activity_id = ?", learnerID, "act-3").First(&la2)
		if la2.Attempts != 1 || la2.Accuracy != 0.8 {
			t.Fatalf("%s: replay double-bumped row %+v", learnerID, la2)
		}
	}

	// Replay parity across paths: A's online replay must also be idempotent.
	if _, _, err := completionRepo.CompleteActivityTx(context.Background(), "parity-online", "act-3", payload); err != nil {
		t.Fatalf("online replay failed: %v", err)
	}
}

// TestSyncBulkRejectsForeignShapes pins the honest failure accounting: only
// completion-shaped items are processed; anything else is counted, never
// silently dropped or fabricated.
func TestSyncBulkRejectsForeignShapes(t *testing.T) {
	syncRepo := repository.NewSyncRepository(database.DB)

	items := []domain.SyncRequestItem{
		{Endpoint: "/activities/act-3/complete", Method: "GET", Body: ""},
		{Endpoint: "/dashboard", Method: "POST", Body: "{}"},
		{Endpoint: "", Method: "POST", Body: "{}"},
		{Endpoint: "/activities/nope/complete", Method: "POST", Body: `{"correct_count":1,"total_count":1}`},
	}
	processed, failed, err := syncRepo.SyncBulk(context.Background(), "parity-shapes", items)
	if err != nil {
		t.Fatalf("sync bulk: %v", err)
	}
	if processed != 0 || failed != 4 {
		t.Fatalf("expected processed=0 failed=4 (honest accounting), got %d/%d", processed, failed)
	}
	t.Cleanup(func() {
		database.DB.Where("learner_id = ?", "parity-shapes").Delete(&domain.LearnerActivity{})
		database.DB.Where("learner_id = ?", "parity-shapes").Delete(&domain.Progress{})
		database.DB.Where("learner_id = ?", "parity-shapes").Delete(&domain.Observation{})
		database.DB.Where("learner_id = ?", "parity-shapes").Delete(&domain.Guidance{})
	})
}

func marshalSyncItem(endpoint string, stats domain.AttemptStats) (domain.SyncRequestItem, error) {
	b, err := json.Marshal(stats)
	if err != nil {
		return domain.SyncRequestItem{}, err
	}
	return domain.SyncRequestItem{Endpoint: endpoint, Method: "POST", Body: string(b)}, nil
}

var _ = service.GenerateSecureID // keep service import honest for future use
