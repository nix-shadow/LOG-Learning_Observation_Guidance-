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

// TestAssignmentPermissions proves the class teacher owns assignment creation
// and learners must be class members to submit.
func TestAssignmentPermissions(t *testing.T) {
	h := newSchoolTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	otherTeacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)
	outsider := newTestUser(t, domain.RoleStudent)

	ctx := context.Background()
	repo := repository.NewSchoolRepository(database.DB)
	class := &domain.Class{ID: service.GenerateSecureID("cls"), Name: "Grade 10 A", Grade: "10", Section: "A", TeacherID: teacher.ID, CreatedAt: time.Now()}
	if err := repo.CreateClass(ctx, class); err != nil {
		t.Fatalf("create class: %v", err)
	}
	if err := repo.Enroll(ctx, class.ID, []string{student.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Where("class_id = ?", class.ID).Delete(&domain.ClassMember{})
		database.DB.Where("class_id = ?", class.ID).Delete(&domain.Assignment{})
		database.DB.Where("id = ?", class.ID).Delete(&domain.Class{})
	})

	r := gin.New()
	actor := otherTeacher.ID
	r.Use(func(c *gin.Context) { c.Set("userID", actor); c.Next() })
	r.POST("/api/v1/moderator/classes/:id/assignments", h.CreateAssignment)
	r.POST("/api/v1/assignments/:assignment_id/submit", h.SubmitAssignment)

	// Non-owner teacher must be rejected with 403
	actor = otherTeacher.ID
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/moderator/classes/"+class.ID+"/assignments", bytes.NewBufferString(`{"title":"Homework 1"}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("other teacher: expected 403, got %v: %s", w.Code, w.Body.String())
	}

	// Owner teacher creates the assignment
	actor = teacher.ID
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/moderator/classes/"+class.ID+"/assignments", bytes.NewBufferString(`{"title":"Homework 1","description":"Solve exercises 1-5"}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create assignment: expected 201, got %v: %s", w.Code, w.Body.String())
	}
	var assignment domain.Assignment
	if err := json.Unmarshal(w.Body.Bytes(), &assignment); err != nil {
		t.Fatalf("unmarshal assignment: %v", err)
	}

	// Class member submits
	actor = student.ID
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/assignments/"+assignment.ID+"/submit", bytes.NewBufferString(`{"note":"My answers are ready"}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("submit: expected 200, got %v: %s", w.Code, w.Body.String())
	}

	// Outsider (not enrolled) must be rejected with 403
	actor = outsider.ID
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/assignments/"+assignment.ID+"/submit", bytes.NewBufferString(`{"note":"sneaky"}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("outsider submit: expected 403, got %v: %s", w.Code, w.Body.String())
	}

	// Teacher sees exactly one submission
	subs, err := repo.SubmissionsForAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("load submissions: %v", err)
	}
	if len(subs) != 1 || subs[0].LearnerID != student.ID {
		t.Fatalf("expected 1 submission from the student, got %+v", subs)
	}
}

// TestSubmissionIdempotent proves resubmitting updates the same row instead of
// duplicating — critical for offline replay integrity.
func TestSubmissionIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewSchoolRepository(database.DB)
	teacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)

	class := &domain.Class{ID: service.GenerateSecureID("cls"), Name: "Grade 11 A", Grade: "11", Section: "A", TeacherID: teacher.ID, CreatedAt: time.Now()}
	if err := repo.CreateClass(ctx, class); err != nil {
		t.Fatalf("create class: %v", err)
	}
	if err := repo.Enroll(ctx, class.ID, []string{student.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	assignment := &domain.Assignment{
		ID:        service.GenerateSecureID("asg"),
		ClassID:   class.ID,
		Title:     "Replay Safe",
		CreatedBy: teacher.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := repo.CreateAssignment(ctx, assignment); err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Where("class_id = ?", class.ID).Delete(&domain.ClassMember{})
		database.DB.Where("class_id = ?", class.ID).Delete(&domain.Assignment{})
		database.DB.Where("id = ?", class.ID).Delete(&domain.Class{})
		database.DB.Where("assignment_id = ?", assignment.ID).Delete(&domain.Submission{})
	})

	h := newSchoolTestHandler(t)
	r := gin.New()
	actor := student.ID
	r.Use(func(c *gin.Context) { c.Set("userID", actor); c.Next() })
	r.POST("/api/v1/assignments/:assignment_id/submit", h.SubmitAssignment)

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/assignments/"+assignment.ID+"/submit", bytes.NewBufferString(`{"note":"v1"}`))
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("submit attempt %d: expected 200, got %v: %s", i+1, w.Code, w.Body.String())
		}
	}

	subs, err := repo.SubmissionsForAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("load submissions: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected exactly 1 submission row after replay, got %d", len(subs))
	}
}
