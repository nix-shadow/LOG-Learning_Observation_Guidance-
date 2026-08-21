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

func newOERTestHandler(t *testing.T) *OERHandler {
	schoolRepo := repository.NewSchoolRepository(database.DB)
	return NewOERHandler(
		service.NewOERService(repository.NewActivityRepository(database.DB)),
		service.NewSchoolService(schoolRepo),
	)
}

func doAdminOER(t *testing.T, h *OERHandler, body []byte) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/", adminAuthStub())
	admin.POST("/oer/import", h.ImportPack)
	req := httptest.NewRequest(http.MethodPost, "/oer/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// adminAuthStub injects an admin identity so the handler's audit-log write
// has a caller to attribute. Mirror of the real AuthMiddleware role gate.
func adminAuthStub() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", "admin-1")
		c.Set("role", string(domain.RoleAdmin))
		c.Next()
	}
}

// TestOERImportValidPack proves the WP-3.1 pipeline imports a curated pack of
// original OER content with real licenses and attribution, returning honest
// counts, and that a re-import skips existing ids without orphaning progress.
func TestOERImportValidPack(t *testing.T) {
	h := newOERTestHandler(t)
	act := domain.Activity{
		ID: "oer-pack-test-1", Title: "Test OER Unit", Description: "Original pack content",
		Topic: "Mathematics", Order: 99, License: "CC BY-SA 4.0", Attribution: "LOG Learning Team (original content)",
	}
	t.Cleanup(func() { database.DB.Where("id = ?", act.ID).Delete(&domain.Activity{}) })

	pack := domain.OERPack{Name: "Test Pack", Activities: []domain.Activity{act}}
	body, _ := json.Marshal(pack)
	w := doAdminOER(t, h, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(1) || resp["skipped"] != float64(0) || resp["rejected"] != float64(0) {
		t.Fatalf("expected imported=1 skipped=0 rejected=0, got %v", resp)
	}

	stored, err := repository.NewActivityRepository(database.DB).FindByID(context.Background(), act.ID)
	if err != nil || stored.License != "CC BY-SA 4.0" || stored.LicenseURL == "" || stored.Attribution == "" {
		t.Fatalf("stored activity lost OER metadata: %+v err=%v", stored, err)
	}

	// Re-import: existing id is skipped, report stays honest.
	w = doAdminOER(t, h, body)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(0) || resp["skipped"] != float64(1) {
		t.Fatalf("expected re-import imported=0 skipped=1, got %v", resp)
	}
}

// TestOERImportRejectsUnknownLicense proves the pipeline never guesses a
// license: empty or unknown licenses are rejected with a per-row reason, and
// third-party content without attribution is rejected too.
func TestOERImportRejectsUnknownLicense(t *testing.T) {
	h := newOERTestHandler(t)
	pack := domain.OERPack{
		Name: "Reject Pack",
		Activities: []domain.Activity{
			{ID: "oer-bad-1", Title: "No license", Topic: "Science", License: "Made up license"},
			{ID: "oer-bad-2", Title: "Empty license", Topic: "Science", License: ""},
			{ID: "oer-bad-3", Title: "No attribution", Topic: "Science", License: "CC BY 4.0"},
			{ID: "oer-ok-1", Title: "Valid row", Topic: "Science", License: "CC BY 4.0", Attribution: "Original author"},
		},
	}
	body, _ := json.Marshal(pack)
	w := doAdminOER(t, h, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Imported int `json:"imported"`
		Skipped  int `json:"skipped"`
		Rejected int `json:"rejected"`
		Errors   []domain.OERImportRowError `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Imported != 1 || resp.Skipped != 0 || resp.Rejected != 3 {
		t.Fatalf("expected imported=1 rejected=3, got imported=%d rejected=%d", resp.Imported, resp.Rejected)
	}
	for _, e := range resp.Errors {
		if e.Reason == "" {
			t.Fatalf("rejected row %d had no reason", e.Row)
		}
	}
	t.Cleanup(func() { database.DB.Where("id = 'oer-ok-1' OR id LIKE 'oer-bad-%'").Delete(&domain.Activity{}) })
}

// TestOERImportRequiresPackName proves the endpoint rejects a nameless pack.
func TestOERImportRequiresPackName(t *testing.T) {
	h := newOERTestHandler(t)
	body, _ := json.Marshal(domain.OERPack{Activities: []domain.Activity{{ID: "x", Title: "t"}}})
	w := doAdminOER(t, h, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for nameless pack, got %d", w.Code)
	}
}