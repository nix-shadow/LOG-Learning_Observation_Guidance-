package handler

import (
	"bytes"
	"context"
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

// TestUnenrollNonMemberReturns404 proves removing a student who is not
// enrolled reports a 404 (Not Found) instead of a misleading success.
func TestUnenrollNonMemberReturns404(t *testing.T) {
	h := newSchoolTestHandler(t)
	admin := newTestUser(t, domain.RoleAdmin)
	teacher := newTestUser(t, domain.RoleModerator)

	ctx := context.Background()
	repo := repository.NewSchoolRepository(database.DB)
	class := &domain.Class{ID: service.GenerateSecureID("cls"), Name: "Grade 10 C", Grade: "10", Section: "C", TeacherID: teacher.ID, CreatedAt: time.Now()}
	if err := repo.CreateClass(ctx, class); err != nil {
		t.Fatalf("create class: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Where("id = ?", class.ID).Delete(&domain.Class{})
	})

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", admin.ID); c.Next() })
	r.DELETE("/api/v1/admin/classes/:id/enroll/:user_id", h.UnenrollStudent)

	// user-123 is enrolled in cls-1, not in this brand-new class
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/admin/classes/"+class.ID+"/enroll/user-123", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-member unenroll, got %v: %s", w.Code, w.Body.String())
	}
}

// TestClassLifecycle proves create -> enroll -> roster -> count -> unenroll
// works end to end and only STUDENT users can be enrolled.
func TestClassLifecycle(t *testing.T) {
	h := newSchoolTestHandler(t)
	admin := newTestUser(t, domain.RoleAdmin)
	teacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)
	staff := newTestUser(t, domain.RoleModerator)

	r := gin.New()
	actor := admin.ID
	r.Use(func(c *gin.Context) { c.Set("userID", actor); c.Next() })
	r.POST("/api/v1/admin/classes", h.CreateClass)
	r.POST("/api/v1/admin/classes/:id/enroll", h.EnrollStudents)
	r.GET("/api/v1/admin/classes/:id/roster", h.ClassRoster)
	r.GET("/api/v1/admin/classes", h.ListClasses)
	r.DELETE("/api/v1/admin/classes/:id/enroll/:user_id", h.UnenrollStudent)

	// Create class
	body := `{"name":"Grade 9 B","grade":"9","section":"B","teacher_id":"` + teacher.ID + `"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/admin/classes", bytes.NewBufferString(body))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create class: expected 201, got %v: %s", w.Code, w.Body.String())
	}
	var class domain.Class
	if err := json.Unmarshal(w.Body.Bytes(), &class); err != nil {
		t.Fatalf("unmarshal class: %v", err)
	}

	// Enroll one STUDENT + one MODERATOR (staff must be ignored)
	body = `{"user_ids":["` + student.ID + `","` + staff.ID + `"]}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/admin/classes/"+class.ID+"/enroll", bytes.NewBufferString(body))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("enroll: expected 200, got %v: %s", w.Code, w.Body.String())
	}
	var enrollResp struct {
		MemberCount int `json:"member_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &enrollResp); err != nil {
		t.Fatalf("unmarshal enroll: %v", err)
	}
	if enrollResp.MemberCount != 1 {
		t.Fatalf("expected 1 enrolled student (staff ignored), got %v", enrollResp.MemberCount)
	}

	// Roster contains exactly the student
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/admin/classes/"+class.ID+"/roster", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("roster: expected 200, got %v", w.Code)
	}
	var rosterResp struct {
		Students []domain.User `json:"students"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rosterResp); err != nil {
		t.Fatalf("unmarshal roster: %v", err)
	}
	if len(rosterResp.Students) != 1 || rosterResp.Students[0].ID != student.ID {
		t.Fatalf("expected exactly the student in roster, got %+v", rosterResp.Students)
	}

	// Unenroll
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/v1/admin/classes/"+class.ID+"/enroll/"+student.ID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unenroll: expected 200, got %v", w.Code)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/admin/classes/"+class.ID+"/roster", nil)
	r.ServeHTTP(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &rosterResp)
	if len(rosterResp.Students) != 0 {
		t.Fatalf("expected empty roster after unenroll, got %+v", rosterResp.Students)
	}
}
