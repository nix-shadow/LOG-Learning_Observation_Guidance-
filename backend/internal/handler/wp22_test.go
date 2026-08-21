package handler

import (
	"bytes"
	"encoding/json"
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

func newSupportTestHandler(t *testing.T) *SupportHandler {
	supportService := service.NewSupportService(repository.NewSupportRepository(database.DB))
	schoolService := service.NewSchoolService(repository.NewSchoolRepository(database.DB))
	return NewSupportHandler(supportService, schoolService)
}

// TestSupportFunnelSelfServed proves the wizard flow: a learner files an
// issue, self-served issues stay out of the inbox, and the issue history is
// honest (WP-2.2 wp22t01).
func TestSupportFunnelSelfServed(t *testing.T) {
	h := newSupportTestHandler(t)
	learner := newTestUser(t, domain.RoleStudent)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", learner.ID); c.Next() })
	r.POST("/api/v1/support/issue", h.CreateIssue)
	r.GET("/api/v1/support/my-issues", h.MyIssues)

	// Self-served (guidance step answered the question): no escalation.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/support/issue", bytes.NewBufferString(`{"category":"connectivity","description":"How do I download modules for offline study?","escalated":false}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %v: %s", w.Code, w.Body.String())
	}
	var issue domain.SupportIssue
	if err := json.Unmarshal(w.Body.Bytes(), &issue); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if issue.Status != domain.SupportStatusOpen || issue.Category != "connectivity" {
		t.Fatalf("unexpected issue: %+v", issue)
	}

	// Unknown category is rejected up front.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/support/issue", bytes.NewBufferString(`{"category":"aliens","description":"This is a long enough description.","escalated":false}`))
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad category, got %v", w2.Code)
	}

	// History lists the issue honestly.
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/v1/support/my-issues", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK || !strings.Contains(w3.Body.String(), issue.ID) {
		t.Fatalf("my-issues must include the issue: %v %s", w3.Code, w3.Body.String())
	}
}

// TestSupportEscalationInbox proves escalated issues land in the moderator
// inbox, resolution is audit-logged, and resolved issues leave the inbox
// (WP-2.2 wp22t02).
func TestSupportEscalationInbox(t *testing.T) {
	h := newSupportTestHandler(t)
	learner := newTestUser(t, domain.RoleStudent)
	moderator := newTestUser(t, domain.RoleModerator)

	// Learner escalates after the guidance step.
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", learner.ID); c.Next() })
	r.POST("/api/v1/support/issue", h.CreateIssue)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/support/issue", bytes.NewBufferString(`{"category":"device","description":"Tablet screen cracked and the app will not start at all.","escalated":true}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %v: %s", w.Code, w.Body.String())
	}
	var issue domain.SupportIssue
	if err := json.Unmarshal(w.Body.Bytes(), &issue); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Moderator inbox sees open escalated issues only.
	rm := gin.New()
	rm.Use(func(c *gin.Context) { c.Set("userID", moderator.ID); c.Next() })
	rm.GET("/api/v1/support/inbox", h.Inbox)
	rm.PUT("/api/v1/support/issue/:id", h.ResolveIssue)

	wm := httptest.NewRecorder()
	reqm, _ := http.NewRequest("GET", "/api/v1/support/inbox", nil)
	rm.ServeHTTP(wm, reqm)
	if wm.Code != http.StatusOK || !strings.Contains(wm.Body.String(), issue.ID) {
		t.Fatalf("inbox must contain the escalated issue: %v %s", wm.Code, wm.Body.String())
	}

	// Resolve with a note — idempotent, audit-logged.
	wr := httptest.NewRecorder()
	reqr, _ := http.NewRequest("PUT", "/api/v1/support/issue/"+issue.ID, bytes.NewBufferString(`{"resolution_note":"Replacement tablet arranged for Monday."}`))
	rm.ServeHTTP(wr, reqr)
	if wr.Code != http.StatusOK {
		t.Fatalf("resolve: %v: %s", wr.Code, wr.Body.String())
	}
	var resolved domain.SupportIssue
	if err := json.Unmarshal(wr.Body.Bytes(), &resolved); err != nil {
		t.Fatalf("unmarshal resolved: %v", err)
	}
	if resolved.Status != domain.SupportStatusResolved || resolved.ResolverID != moderator.ID {
		t.Fatalf("unexpected resolution: %+v", resolved)
	}

	// Resolved issues leave the inbox — no stale work.
	wm2 := httptest.NewRecorder()
	reqm2, _ := http.NewRequest("GET", "/api/v1/support/inbox", nil)
	rm.ServeHTTP(wm2, reqm2)
	if strings.Contains(wm2.Body.String(), issue.ID) {
		t.Fatalf("resolved issue must leave the inbox: %s", wm2.Body.String())
	}

	// Every action is audit-logged.
	var created, resolvedAudit int64
	database.DB.Model(&domain.AuditLog{}).Where("action = ?", "support.issue_created").Count(&created)
	database.DB.Model(&domain.AuditLog{}).Where("action = ?", "support.issue_resolved").Count(&resolvedAudit)
	if created == 0 || resolvedAudit == 0 {
		t.Fatalf("expected audit rows for issue_created and issue_resolved, got %d/%d", created, resolvedAudit)
	}

	// Unknown issue → honest 404.
	wn := httptest.NewRecorder()
	reqn, _ := http.NewRequest("PUT", "/api/v1/support/issue/does-not-exist", bytes.NewBufferString(`{"resolution_note":"nope"}`))
	rm.ServeHTTP(wn, reqn)
	if wn.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown issue, got %v", wn.Code)
	}
}
