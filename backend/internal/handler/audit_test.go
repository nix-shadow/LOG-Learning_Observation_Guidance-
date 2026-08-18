package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
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
	r.Use(func(c *gin.Context) { c.Set("userID", actor); c.Next() })
	r.PUT("/api/v1/admin/users/:id/role", UpdateUserRole)

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
