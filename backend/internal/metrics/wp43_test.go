package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMiddlewareRecordsRoutePatternAndStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRegistry()
	engine := gin.New()
	engine.Use(Middleware(r))
	engine.GET("/api/v1/dashboard", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	engine.GET("/api/v1/users/:id/role", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/dashboard", nil))
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/users/abc-123/role", nil))

	snap := r.Snapshot()
	if snap.TotalRequests != 4 {
		t.Fatalf("TotalRequests = %d, want 4", snap.TotalRequests)
	}

	// The dynamic route must be recorded as its PATTERN, never the raw id.
	found := false
	for _, rs := range snap.Routes {
		if rs.Route == "/api/v1/users/:id/role" {
			found = true
			if rs.Method != "GET" || rs.Total != 1 || rs.ByStatus[200] != 1 {
				t.Fatalf("route counters wrong: %+v", rs)
			}
		}
		if rs.Route == "/api/v1/users/abc-123/role" {
			t.Fatalf("raw URL leaked into route dimension: %q", rs.Route)
		}
	}
	if !found {
		t.Fatal("pattern route not recorded")
	}
}

func TestSpikeAlarmFiresOnceAndRearms(t *testing.T) {
	r := NewRegistry()

	// 4 five-hundreds: under threshold, no alarm.
	for i := 0; i < 4; i++ {
		r.Record("GET", "/api/v1/dashboard", 500)
	}
	snap := r.Snapshot()
	if snap.SpikeActive {
		t.Fatal("alarm fired below threshold")
	}

	// 5th within the window trips it; further 5xx must not re-fire (last-alert state).
	r.Record("GET", "/api/v1/dashboard", 500)
	snap = r.Snapshot()
	if !snap.SpikeActive {
		t.Fatal("alarm did not fire at threshold")
	}
	if snap.LastAlert == nil {
		t.Fatal("LastAlert not recorded")
	}
	firstAlert := *snap.LastAlert

	r.Record("GET", "/api/v1/dashboard", 500)
	snap = r.Snapshot()
	if *snap.LastAlert != firstAlert {
		t.Fatal("alarm re-fired during active spike — last-alert state violated")
	}

	// Window slides out: alarm re-arms.
	// Simulate the window passing by expiring recorded timestamps directly.
	r.mu.Lock()
	r.spikeAt = []time.Time{time.Now().Add(-2 * SpikeWindow)}
	r.mu.Unlock()
	snap = r.Snapshot()
	if snap.SpikeActive {
		t.Fatal("alarm still active after window expired")
	}
	// A fresh burst re-fires.
	for i := 0; i < SpikeThreshold; i++ {
		r.Record("POST", "/api/v1/sync/bulk", 503)
	}
	snap = r.Snapshot()
	if !snap.SpikeActive {
		t.Fatal("alarm did not re-arm after window cleared")
	}
}

func TestRenderTextCarriesNoPII(t *testing.T) {
	r := NewRegistry()
	r.Record("POST", "/api/v1/users/:id/role", 200)
	r.Record("POST", "/api/v1/users/:id/role", 403)
	text := r.RenderText()

	if !strings.Contains(text, `route="/api/v1/users/:id/role"`) {
		t.Fatalf("rendered output missing route pattern:\n%s", text)
	}
	if !strings.Contains(text, `status="403"`) {
		t.Fatalf("rendered output missing status counter:\n%s", text)
	}
	if strings.Contains(text, "/api/v1/abc-123") || strings.Contains(text, "@") {
		t.Fatalf("rendered output may contain PII:\n%s", text)
	}
	if !strings.HasPrefix(text, "# TYPE log_http_requests_total counter") {
		t.Fatalf("unexpected text format:\n%s", text)
	}
}

func TestUnmatchedRoutesAreNotCounted(t *testing.T) {
	r := NewRegistry()
	r.Record("GET", "", 404)
	snap := r.Snapshot()
	if snap.TotalRequests != 0 {
		t.Fatalf("empty pattern counted: %d", snap.TotalRequests)
	}
}
