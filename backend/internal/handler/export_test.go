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

// TestStudentsCSVExportSanitizesFormulaCells proves spreadsheet formula
// injection is neutralized: cells starting with = + - @ tab or CR get a
// leading apostrophe, and the file ships with a UTF-8 BOM.
func TestStudentsCSVExportSanitizesFormulaCells(t *testing.T) {
	h := newSchoolTestHandler(t)
	admin := newTestUser(t, domain.RoleAdmin)
	teacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)

	// Hostile-but-plausible learner data: a formula name, a +977 phone
	student.Name = "=SUM(A1:A9)"
	plusPhone := "+977" + service.GenerateSecureID("ph")[3:] // unique, phone-like
	student.Phone = &plusPhone
	database.DB.Model(&domain.User{}).Where("id = ?", student.ID).Updates(map[string]interface{}{"name": student.Name, "phone": plusPhone})

	ctx := context.Background()
	repo := repository.NewSchoolRepository(database.DB)
	class := &domain.Class{ID: service.GenerateSecureID("cls"), Name: "Grade 8 B", Grade: "8", Section: "B", TeacherID: teacher.ID, CreatedAt: time.Now()}
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
	r.Use(func(c *gin.Context) { c.Set("userID", admin.ID); c.Next() })
	r.GET("/api/v1/admin/export/students.csv", h.ExportStudentsCSV)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/export/students.csv", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("export: expected 200, got %v", w.Code)
	}
	body := w.Body.String()

	// UTF-8 BOM must open the file
	if !strings.HasPrefix(body, "\xEF\xBB\xBF") {
		t.Fatalf("expected UTF-8 BOM prefix, got %q", body[:3])
	}

	// Formula cells are neutralized with a leading apostrophe
	if !strings.Contains(body, "'=SUM(A1:A9)") {
		t.Fatalf("expected sanitized formula cell, got: %s", body)
	}
	if !strings.Contains(body, "'+977") {
		t.Fatalf("expected sanitized phone cell, got: %s", body)
	}
	// Raw formulas must never reach the file
	if strings.Contains(strings.TrimPrefix(body, "\xEF\xBB\xBF"), "\n=SUM(A1:A9)") {
		t.Fatalf("raw formula leaked into CSV: %s", body)
	}
}

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
