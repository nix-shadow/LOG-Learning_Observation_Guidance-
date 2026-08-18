package handler

import (
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

// TestStudentsCSVExportHonest proves the export contains exactly the real
// enrolled students and no fabricated rows.
func TestStudentsCSVExportHonest(t *testing.T) {
	h := newSchoolTestHandler(t)
	admin := newTestUser(t, domain.RoleAdmin)
	teacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)

	ctx := context.Background()
	repo := repository.NewSchoolRepository(database.DB)
	class := &domain.Class{ID: service.GenerateSecureID("cls"), Name: "Grade 8 A", Grade: "8", Section: "A", TeacherID: teacher.ID, CreatedAt: time.Now()}
	if err := repo.CreateClass(ctx, class); err != nil {
		t.Fatalf("create class: %v", err)
	}
	if err := repo.Enroll(ctx, class.ID, []string{student.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Where("class_id = ?", class.ID).Delete(&domain.ClassMember{})
		database.DB.Where("id = ?", class.ID).Delete(&domain.Class{})
	})

	r := gin.New()
	actor := admin.ID
	r.Use(func(c *gin.Context) { c.Set("userID", actor); c.Next() })
	r.GET("/api/v1/admin/export/students.csv", h.ExportStudentsCSV)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/export/students.csv", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("export: expected 200, got %v", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "name,email,phone,class") {
		t.Fatalf("expected header row, got: %s", body)
	}
	if !strings.Contains(body, student.Email) {
		t.Fatalf("expected enrolled student in export, got: %s", body)
	}
	if strings.Contains(body, teacher.Email) {
		t.Fatalf("staff must never appear in student export: %s", body)
	}
}
