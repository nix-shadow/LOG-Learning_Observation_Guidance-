package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// setupAdminRoleRouter mounts the AdminHandler (C3 seam) role endpoint with a
// stubbed actor identity.
func setupAdminRoleRouter(actorID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(service.NewAdminService(repository.NewAdminRepository(database.DB)))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", actorID)
		c.Next()
	})
	r.PUT("/api/v1/admin/users/:id/role", h.UpdateUserRole)
	return r
}

func seedUser(t *testing.T, id string, role domain.Role) {
	t.Helper()
	phone := service.GenerateSecureID("ph")
	database.DB.Create(&domain.User{
		ID: id, Name: "Guard Test", Email: id + "@guard.edu",
		Phone: &phone, Role: role, IsVerified: true,
	})
	t.Cleanup(func() {
		database.DB.Where("id = ?", id).Delete(&domain.User{})
		database.DB.Where("user_id = ?", id).Delete(&domain.AuditLog{})
	})
}

func demoteUser(t *testing.T, r *gin.Engine, targetID string, role domain.Role) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"role": string(role)})
	req, _ := http.NewRequest("PUT", "/api/v1/admin/users/"+targetID+"/role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestUpdateUserRoleLastAdminGuard proves the final admin cannot be demoted —
// neither by another admin nor by themselves — while demotions that leave at
// least one principal (and same-role no-ops) still succeed.
func TestUpdateUserRoleLastAdminGuard(t *testing.T) {
	seedUser(t, "admin-guard-2", domain.RoleAdmin)
	r := setupAdminRoleRouter("admin-guard-2")

	// Two admins exist (admin-1 from seed + admin-guard-2): demotion allowed.
	if w := demoteUser(t, r, "admin-guard-2", domain.RoleModerator); w.Code != http.StatusOK {
		t.Fatalf("expected 200 demoting with another admin remaining, got %v: %s", w.Code, w.Body.String())
	}
	var demoted domain.User
	database.DB.Where("id = ?", "admin-guard-2").First(&demoted)
	if demoted.Role != domain.RoleModerator {
		t.Fatalf("expected role MODERATOR, got %q", demoted.Role)
	}

	// Now admin-1 is the last admin: any demotion must be rejected.
	if w := demoteUser(t, r, "admin-1", domain.RoleModerator); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 demoting the last admin, got %v: %s", w.Code, w.Body.String())
	}
	// NOTE: a fresh struct is required here — GORM reuses a populated
	// primary key from a previously-read dest, silently ANDing both ids.
	var lastAdmin domain.User
	database.DB.Where("id = ?", "admin-1").First(&lastAdmin)
	if lastAdmin.Role != domain.RoleAdmin {
		t.Fatalf("last admin must keep ADMIN role, got %q", lastAdmin.Role)
	}

	// Rejected changes must not pollute the audit trail.
	var auditCount int64
	database.DB.Model(&domain.AuditLog{}).Where("user_id = ? AND action = ?", "admin-1", "user.role_change").Count(&auditCount)
	if auditCount != 0 {
		t.Fatalf("rejected demotion must not be audited, got %d rows", auditCount)
	}

	// Self-demotion of the last admin is equally blocked.
	if w := demoteUser(t, r, "admin-1", domain.RoleStudent); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 self-demoting the last admin, got %v", w.Code)
	}

	// Same-role no-op on the last admin is fine.
	if w := demoteUser(t, r, "admin-1", domain.RoleAdmin); w.Code != http.StatusOK {
		t.Fatalf("expected 200 same-role no-op, got %v: %s", w.Code, w.Body.String())
	}

	// Demoting a non-admin is never blocked by the guard.
	seedUser(t, "student-guard-3", domain.RoleStudent)
	if w := demoteUser(t, r, "student-guard-3", domain.RoleModerator); w.Code != http.StatusOK {
		t.Fatalf("expected 200 demoting a student, got %v: %s", w.Code, w.Body.String())
	}
}

// TestAnalyticsSummaryOptInGate (WP-4.3) proves the aggregate view only ever
// includes learners with an active analytics consent. Two learners complete
// the same activity; only one opts in — the summary must count exactly that
// learner's work, never the other's.
func TestAnalyticsSummaryOptInGate(t *testing.T) {
	phone := service.GenerateSecureID("ph")
	learnerA := &domain.User{ID: "analytics-a", Name: "Opted In", Email: "analytics-a@gate.edu", Phone: &phone, Role: domain.RoleStudent, IsVerified: true}
	phoneB := service.GenerateSecureID("ph")
	learnerB := &domain.User{ID: "analytics-b", Name: "Not Opted", Email: "analytics-b@gate.edu", Phone: &phoneB, Role: domain.RoleStudent, IsVerified: true}
	database.DB.Create(learnerA)
	database.DB.Create(learnerB)
	t.Cleanup(func() {
		database.DB.Where("id IN ?", []string{"analytics-a", "analytics-b"}).Delete(&domain.User{})
		database.DB.Where("learner_id IN ?", []string{"analytics-a", "analytics-b"}).Delete(&domain.LearnerActivity{})
		database.DB.Where("user_id IN ?", []string{"analytics-a", "analytics-b"}).Delete(&domain.ConsentRecord{})
	})

	// Both learners completed the same activity — identical usage. A real
	// activity row is created so the C4 activity_id FK accepts the inserts.
	act := &domain.Activity{ID: "act-analytics-1", Title: "Analytics Gate", Topic: "test", Difficulty: "beginner", CreatedAt: time.Now()}
	database.DB.Create(act)
	t.Cleanup(func() {
		database.DB.Unscoped().Where("id = ?", act.ID).Delete(&domain.Activity{})
	})

	completed := domain.StatusCompleted
	for _, lid := range []string{"analytics-a", "analytics-b"} {
		database.DB.Create(&domain.LearnerActivity{
			LearnerID:   lid,
			ActivityID:  "act-analytics-1",
			Status:      completed,
			Score:       70.0,
			CompletedAt: time.Now(),
		})
	}
	// Only learner A grants analytics consent.
	database.DB.Create(&domain.ConsentRecord{
		ID:          service.GenerateSecureID("csn"),
		UserID:      "analytics-a",
		ConsentType: domain.ConsentTypeAnalytics,
		Version:     domain.PolicyVersion,
		Status:      domain.ConsentStatusGranted,
		GrantedBy:   "self",
		Language:    "en",
		Source:      "settings",
	})

	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(service.NewAdminService(repository.NewAdminRepository(database.DB)))
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "admin-1"); c.Next() })
	r.GET("/api/v1/admin/analytics/summary", h.AnalyticsSummary)

	req, _ := http.NewRequest("GET", "/api/v1/admin/analytics/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %v: %s", w.Code, w.Body.String())
	}

	var sum domain.AnalyticsSummary
	if err := json.Unmarshal(w.Body.Bytes(), &sum); err != nil {
		t.Fatalf("bad summary JSON: %v", err)
	}
	if sum.OptedInUsers != 1 {
		t.Fatalf("OptedInUsers = %d, want 1", sum.OptedInUsers)
	}
	if sum.Completions != 1 {
		t.Fatalf("Completions = %d, want 1 (only opted-in learner counts)", sum.Completions)
	}
	if sum.AvgScore == nil || *sum.AvgScore != 70.0 {
		t.Fatalf("AvgScore = %v, want 70.0", sum.AvgScore)
	}
	if sum.ActiveDaily != 1 {
		t.Fatalf("ActiveDaily = %d, want 1", sum.ActiveDaily)
	}
}
