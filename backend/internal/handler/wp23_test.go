package handler

import (
	"bytes"
	"context"
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

func newGradebookTestHandler(t *testing.T) *GradebookHandler {
	schoolRepo := repository.NewSchoolRepository(database.DB)
	gradebookService := service.NewGradebookService(
		schoolRepo,
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewNoteRepository(database.DB),
	)
	return NewGradebookHandler(gradebookService, service.NewSchoolService(schoolRepo))
}

// TestGradebookHonest proves every gradebook number is a real stored
// LearnerActivity row: a learner with no rows shows the honest empty state
// (frontend renders "Not yet assessed"), accuracy/attempts are real, and the
// CSV exports only real data (WP-2.3 wp23t01).
func TestGradebookHonest(t *testing.T) {
	h := newGradebookTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	other := newTestUser(t, domain.RoleModerator)
	assessed := newTestUser(t, domain.RoleStudent)
	unassessed := newTestUser(t, domain.RoleStudent)
	class := newClassWithTeacher(t, teacher.ID)
	if err := repository.NewSchoolRepository(database.DB).Enroll(context.Background(), class.ID, []string{assessed.ID, unassessed.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	enrollCleanup(t, class.ID, assessed.ID)
	enrollCleanup(t, class.ID, unassessed.ID)

	act := &domain.Activity{ID: service.GenerateSecureID("act"), Title: "Nepali Essay", Topic: "Nepali"}
	if err := database.DB.Create(act).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	t.Cleanup(func() { database.DB.Where("id = ?", act.ID).Delete(&domain.Activity{}) })
	database.DB.Create(&domain.LearnerActivity{
		LearnerID:  assessed.ID,
		ActivityID: act.ID,
		Status:     domain.StatusNeedsPractice,
		Accuracy:   0.6,
		Attempts:   3,
	})
	t.Cleanup(func() { database.DB.Where("learner_id = ?", assessed.ID).Delete(&domain.LearnerActivity{}) })

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", teacher.ID); c.Next() })
	r.GET("/api/v1/moderator/gradebook", h.ClassGradebook)
	r.GET("/api/v1/moderator/gradebook.csv", h.GradebookCSV)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/moderator/gradebook?class_id="+class.ID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("gradebook: %v: %s", w.Code, w.Body.String())
	}
	var payload struct {
		Students []struct {
			StudentID string `json:"student_id"`
			Name      string `json:"name"`
			Rows      []struct {
				Title    string  `json:"title"`
				Status   string  `json:"status"`
				Accuracy float64 `json:"accuracy"`
				Attempts int     `json:"attempts"`
			} `json:"rows"`
		} `json:"students"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var assessedRow, unassessedRow int
	for _, s := range payload.Students {
		switch s.StudentID {
		case assessed.ID:
			assessedRow = len(s.Rows)
			if assessedRow == 0 || s.Rows[0].Status != domain.StatusNeedsPractice {
				t.Fatalf("assessed learner must carry the real status: %+v", s)
			}
			if s.Rows[0].Accuracy != 0.6 || s.Rows[0].Attempts != 3 {
				t.Fatalf("real accuracy/attempts must be stored: %+v", s.Rows[0])
			}
		case unassessed.ID:
			unassessedRow = len(s.Rows)
			if unassessedRow == 0 {
				t.Fatalf("unassessed learner must still appear (empty rows = honest 'Not yet assessed'): %+v", s)
			}
			if s.Rows[0].Accuracy != 0 || s.Rows[0].Attempts != 0 {
				t.Fatalf("no invented numbers: %+v", s.Rows[0])
			}
		}
	}
	if assessedRow == 0 || unassessedRow == 0 {
		t.Fatalf("both learners must appear in the gradebook")
	}

	// CSV: real data only, sanitized, no fabricated grades.
	wc := httptest.NewRecorder()
	reqc, _ := http.NewRequest("GET", "/api/v1/moderator/gradebook.csv?class_id="+class.ID, nil)
	r.ServeHTTP(wc, reqc)
	if wc.Code != http.StatusOK {
		t.Fatalf("csv: %v", wc.Code)
	}
	csvBody := wc.Body.String()
	if !strings.Contains(csvBody, "student_id,student_name,activity_id,title,topic,status,accuracy,attempts") {
		t.Fatalf("csv header missing: %s", csvBody)
	}
	if !strings.Contains(csvBody, "needs-practice") || !strings.Contains(csvBody, "0.6") || !strings.Contains(csvBody, "3") {
		t.Fatalf("csv must carry real data: %s", csvBody)
	}

	// Non-owner teacher → hard 404.
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("userID", other.ID); c.Next() })
	r2.GET("/api/v1/moderator/gradebook", h.ClassGradebook)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/moderator/gradebook?class_id="+class.ID, nil)
	r2.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner, got %v", w2.Code)
	}
}

// TestLearnerNotes proves the one-editable-note contract with honest null
// before any note exists, scope enforcement, and the upsert (WP-2.3
// wp23t02).
func TestLearnerNotes(t *testing.T) {
	h := newGradebookTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	other := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)
	class := newClassWithTeacher(t, teacher.ID)
	if err := repository.NewSchoolRepository(database.DB).Enroll(context.Background(), class.ID, []string{student.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	enrollCleanup(t, class.ID, student.ID)
	t.Cleanup(func() { database.DB.Where("student_id = ?", student.ID).Delete(&domain.LearnerNote{}) })

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", teacher.ID); c.Next() })
	r.GET("/api/v1/moderator/students/:id/note", h.GetNote)
	r.PUT("/api/v1/moderator/students/:id/note", h.SaveNote)

	// Honest null before any note exists.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/moderator/students/"+student.ID+"/note", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"note":null`) {
		t.Fatalf("expected honest null note, got %v: %s", w.Code, w.Body.String())
	}

	// Upsert a supportive note.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("PUT", "/api/v1/moderator/students/"+student.ID+"/note", bytes.NewBufferString(`{"note":"Great improvement this week on essay structure!"}`))
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("save note: %v: %s", w2.Code, w2.Body.String())
	}
	var saved struct {
		Note string `json:"note"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &saved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if saved.Note != "Great improvement this week on essay structure!" {
		t.Fatalf("unexpected note: %q", saved.Note)
	}

	// Second save overwrites (one editable note, upsert).
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("PUT", "/api/v1/moderator/students/"+student.ID+"/note", bytes.NewBufferString(`{"note":"Now focusing on revision practice."}`))
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("update note: %v", w3.Code)
	}
	var notes int64
	database.DB.Model(&domain.LearnerNote{}).Where("student_id = ?", student.ID).Count(&notes)
	if notes != 1 {
		t.Fatalf("expected exactly 1 note row after upsert, got %d", notes)
	}

	// Non-owner teacher cannot read or write notes.
	r4 := gin.New()
	r4.Use(func(c *gin.Context) { c.Set("userID", other.ID); c.Next() })
	r4.GET("/api/v1/moderator/students/:id/note", h.GetNote)
	r4.PUT("/api/v1/moderator/students/:id/note", h.SaveNote)
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/api/v1/moderator/students/"+student.ID+"/note", nil)
	r4.ServeHTTP(w4, req4)
	if w4.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner read, got %v", w4.Code)
	}
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest("PUT", "/api/v1/moderator/students/"+student.ID+"/note", bytes.NewBufferString(`{"note":"should not save"}`))
	r4.ServeHTTP(w5, req5)
	if w5.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner write, got %v", w5.Code)
	}
}
