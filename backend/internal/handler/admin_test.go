package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// setupAdminRoleRouter mounts UpdateUserRole with a stubbed actor identity.
func setupAdminRoleRouter(actorID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", actorID)
		c.Next()
	})
	r.PUT("/api/v1/admin/users/:id/role", UpdateUserRole)
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
