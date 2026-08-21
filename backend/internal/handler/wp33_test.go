package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func newPilotTestHandler(t *testing.T) *PilotHandler {
	pilotRepo := repository.NewPilotRepository(database.DB)
	activityRepo := repository.NewActivityRepository(database.DB)
	return NewPilotHandler(service.NewPilotService(pilotRepo, func(ctx context.Context, activityID string) (bool, error) {
		_, err := activityRepo.FindByID(ctx, activityID)
		if err != nil {
			return false, nil
		}
		return true, nil
	}))
}

func doPilotScan(t *testing.T, h *PilotHandler, posterID string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/pilot/scans", h.RecordScan)
	r.POST("/pilot/scans/:id/start", h.MarkStarted)
	body, _ := json.Marshal(map[string]string{"poster_id": posterID, "source": "qr"})
	req := httptest.NewRequest(http.MethodPost, "/pilot/scans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestPilotRecordScan proves a QR poster scan is recorded for a real activity
// and returns its scan id so the landing page can later mark a start.
func TestPilotRecordScan(t *testing.T) {
	h := newPilotTestHandler(t)
	act := &domain.Activity{ID: "pilot-poster-act", Title: "Pilot Poster", Topic: "Science", License: "Own work (LOG team)", Attribution: "LOG Learning Team"}
	if err := database.DB.Create(act).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Where("id = ?", act.ID).Delete(&domain.Activity{})
		database.DB.Where("poster_id = ?", act.ID).Delete(&domain.PilotScan{})
	})

	w := doPilotScan(t, h, act.ID)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		ScanID uint `json:"scan_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.ScanID == 0 {
		t.Fatalf("expected a scan id, got %s (err=%v)", w.Body.String(), err)
	}

	var scan domain.PilotScan
	if err := database.DB.First(&scan, resp.ScanID).Error; err != nil {
		t.Fatalf("scan not persisted: %v", err)
	}
	if scan.PosterID != act.ID || scan.Source != "qr" || scan.Started {
		t.Fatalf("scan row mismatch: %+v", scan)
	}
}

// TestPilotRejectsUnknownPoster proves a scan against a non-existent activity
// is a 404, not a silently recorded fabricated poster.
func TestPilotRejectsUnknownPoster(t *testing.T) {
	h := newPilotTestHandler(t)
	w := doPilotScan(t, h, "no-such-activity")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// TestPilotStartThenStats proves the honest first-session drop-off: a scan
// that never clicks through contributes to scans but not starts, and the
// start rate is derived from real rows (0 when nothing started).
func TestPilotStartThenStats(t *testing.T) {
	h := newPilotTestHandler(t)
	act := &domain.Activity{ID: "pilot-poster-2", Title: "Pilot Poster 2", Topic: "Maths", License: "Own work (LOG team)", Attribution: "LOG Learning Team"}
	if err := database.DB.Create(act).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Where("id = ?", act.ID).Delete(&domain.Activity{})
		database.DB.Where("poster_id = ?", act.ID).Delete(&domain.PilotScan{})
	})

	// Two scans, one click-through.
	first := doPilotScan(t, h, act.ID)
	doPilotScan(t, h, act.ID)

	var resp struct {
		ScanID uint `json:"scan_id"`
	}
	_ = json.Unmarshal(first.Body.Bytes(), &resp)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/pilot/scans/:id/start", h.MarkStarted)
	req := httptest.NewRequest(http.MethodPost, "/pilot/scans/"+uintStr(resp.ScanID)+"/start", nil)
	startW := httptest.NewRecorder()
	r.ServeHTTP(startW, req)
	if startW.Code != http.StatusOK {
		t.Fatalf("expected 200 on start, got %d", startW.Code)
	}

	stats, err := h.pilotService.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalScans != 2 || stats.Starts != 1 {
		t.Fatalf("expected scans=2 starts=1, got %+v", stats)
	}
	if stats.StartRate != 0.5 {
		t.Fatalf("expected start rate 0.5, got %v", stats.StartRate)
	}
	if len(stats.PerPoster) == 0 || stats.PerPoster[0].PosterID != act.ID {
		t.Fatalf("per-poster breakdown missing: %+v", stats.PerPoster)
	}
}

// TestPilotStatsHonestZeros proves an empty pilot reports real zeros, never
// invented numbers (AGENTS.md §1).
func TestPilotStatsHonestZeros(t *testing.T) {
	h := newPilotTestHandler(t)
	stats, err := h.pilotService.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalScans != 0 || stats.Starts != 0 || stats.StartRate != 0 || stats.DistinctPosters != 0 {
		t.Fatalf("expected honest zeros, got %+v", stats)
	}
}

func uintStr(n uint) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}