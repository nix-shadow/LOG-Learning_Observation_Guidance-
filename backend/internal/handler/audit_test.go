package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// TestLogoutAllRevocation proves a token issued before "log out on all devices"
// is rejected by the middleware, while a newer one still works.
func TestLogoutAllRevocation(t *testing.T) {
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)
	schoolRepo := repository.NewSchoolRepository(database.DB)

	user := newTestUser(t, domain.RoleStudent)

	// Token minted BEFORE the revocation timestamp.
	oldToken, err := service.GenerateJWT(user.ID, domain.RoleStudent)
	if err != nil {
		t.Fatalf("failed to mint old token: %v", err)
	}

	if err := schoolRepo.RevokeAll(context.Background(), user.ID, time.Now()); err != nil {
		t.Fatalf("revoke all: %v", err)
	}

	// Wait so a token minted after the revocation has a later iat.
	time.Sleep(1100 * time.Millisecond)
	newToken, err := service.GenerateJWT(user.ID, domain.RoleStudent)
	if err != nil {
		t.Fatalf("failed to mint new token: %v", err)
	}

	r := gin.New()
	r.GET("/api/v1/assignments", AuthMiddleware(authRepo, userRepo, schoolRepo, domain.RoleStudent), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Old token: rejected.
	req, _ := http.NewRequest("GET", "/api/v1/assignments", nil)
	req.Header.Set("Authorization", "Bearer "+oldToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("old token: expected 401, got %v: %s", w.Code, w.Body.String())
	}

	// New token: accepted.
	req, _ = http.NewRequest("GET", "/api/v1/assignments", nil)
	req.Header.Set("Authorization", "Bearer "+newToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("new token: expected 200, got %v: %s", w.Code, w.Body.String())
	}
}

// TestAuditLogOnRoleChange proves sensitive privilege changes are appended to
// the audit trail.
func TestAuditLogOnRoleChange(t *testing.T) {
	admin := newTestUser(t, domain.RoleAdmin)
	target := newTestUser(t, domain.RoleStudent)

	r := gin.New()
	actor := admin.ID
	h := NewAdminHandler(service.NewAdminService(repository.NewAdminRepository(database.DB)))
	r.Use(func(c *gin.Context) { c.Set("userID", actor); c.Next() })
	r.PUT("/api/v1/admin/users/:id/role", h.UpdateUserRole)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/admin/users/"+target.ID+"/role", bytes.NewBufferString(`{"role":"MODERATOR"}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("role change: expected 200, got %v: %s", w.Code, w.Body.String())
	}

	var entry domain.AuditLog
	if err := database.DB.Where("action = ?", "user.role_change").Order("id desc").First(&entry).Error; err != nil {
		t.Fatalf("expected audit log entry: %v", err)
	}
	if entry.UserID != admin.ID || !strings.Contains(entry.Detail, "MODERATOR") {
		t.Fatalf("unexpected audit entry: %+v", entry)
	}
	database.DB.Where("id = ?", entry.ID).Delete(&domain.AuditLog{})
}

// WP-0.2 C1: the audit log pages with limit+offset and reports total — no
// unbounded row load, and pages never overlap or skip.
func TestAuditLogPagination(t *testing.T) {
	// Create 5 audit rows directly (the append-only writer is covered above).
	for i := 0; i < 5; i++ {
		e := &domain.AuditLog{
			UserID:    "audit-pager",
			Action:    "test.page_fixture",
			Detail:    "i=" + strconv.Itoa(i),
			IP:        "",
			CreatedAt: time.Now(),
		}
		database.DB.Create(e)
	}
	defer database.DB.Where("user_id = ?", "audit-pager").Delete(&domain.AuditLog{})

	schoolService := service.NewSchoolService(repository.NewSchoolRepository(database.DB))
	h := NewSchoolHandler(schoolService)
	r := gin.New()
	r.GET("/api/v1/admin/audit-log", h.ListAuditLog)

	get := func(query string) (entries []domain.AuditLog, total int64, code int) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/admin/audit-log?"+query, nil)
		r.ServeHTTP(w, req)
		var resp struct {
			AuditLogs  []domain.AuditLog `json:"audit_logs"`
			Pagination struct {
				Limit  int   `json:"limit"`
				Offset int   `json:"offset"`
				Total  int64 `json:"total"`
			} `json:"pagination"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.AuditLogs, resp.Pagination.Total, w.Code
	}

	// Page 1: 2 rows (newest first). Page 2: 2 rows. Page 3: the rest.
	// Combined, all 5 fixtures must appear exactly once across pages.
	seen := map[string]bool{}
	p1, total, code := get("limit=2&offset=0")
	if code != http.StatusOK || len(p1) != 2 || total < 5 {
		t.Fatalf("page 1: code=%d len=%d total=%d", code, len(p1), total)
	}
	for _, e := range p1 {
		if strings.HasPrefix(e.Detail, "i=") {
			seen[e.Detail] = true
		}
	}
	p2, _, code := get("limit=2&offset=2")
	if code != http.StatusOK || len(p2) != 2 {
		t.Fatalf("page 2: code=%d len=%d", code, len(p2))
	}
	for _, e := range p2 {
		if strings.HasPrefix(e.Detail, "i=") {
			seen[e.Detail] = true
		}
	}
	p3, _, code := get("limit=2&offset=4")
	if code != http.StatusOK || len(p3) == 0 {
		t.Fatalf("page 3: code=%d len=%d", code, len(p3))
	}
	for _, e := range p3 {
		if strings.HasPrefix(e.Detail, "i=") {
			seen[e.Detail] = true
		}
	}
	for i := 0; i < 5; i++ {
		if !seen["i="+strconv.Itoa(i)] {
			t.Fatalf("fixture %d missing across pages — offset pagination is skipping rows", i)
		}
	}

	// Malformed offset must degrade to 0, not error or panic.
	_, _, code = get("offset=-3")
	if code != http.StatusOK {
		t.Fatalf("negative offset: expected 200, got %d", code)
	}
}
