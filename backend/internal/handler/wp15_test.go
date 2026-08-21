package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
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

// newModeratorProgressHandler wires the LearnerHandler the way main.go does
// so the per-student progress route can be exercised end to end.
func newModeratorProgressHandler(t *testing.T) *LearnerHandler {
	userRepo := repository.NewUserRepository(database.DB)
	activityRepo := repository.NewActivityRepository(database.DB)
	progressRepo := repository.NewProgressRepository(database.DB)
	learnerDataRepo := repository.NewLearnerDataRepository(database.DB)
	completionRepo := repository.NewCompletionRepository(database.DB)
	learnerService := service.NewLearnerService(userRepo, activityRepo, progressRepo, learnerDataRepo, completionRepo)
	schoolRepo := repository.NewSchoolRepository(database.DB)
	schoolService := service.NewSchoolService(schoolRepo)
	return NewLearnerHandler(learnerService, nil, nil, schoolService)
}

func newClassWithTeacher(t *testing.T, teacherID string) *domain.Class {
	repo := repository.NewSchoolRepository(database.DB)
	class := &domain.Class{
		ID:         service.GenerateSecureID("cls"),
		Name:       "Grade 9 A",
		Grade:      "9",
		Section:    "A",
		TeacherID:  teacherID,
		InviteCode: service.GenerateInviteCode(),
		CreatedAt:  time.Now(),
	}
	if err := repo.CreateClass(context.Background(), class); err != nil {
		t.Fatalf("create class: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Where("id = ?", class.ID).Delete(&domain.Class{})
	})
	return class
}

func enrollCleanup(t *testing.T, classID, userID string) {
	t.Cleanup(func() {
		database.DB.Where("class_id = ? AND user_id = ?", classID, userID).Delete(&domain.ClassMember{})
	})
}

// TestCreateModeratorClass proves a teacher can create a class and receives
// a joinable invite code (WP-1.5 wizard).
func TestCreateModeratorClass(t *testing.T) {
	h := newSchoolTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", teacher.ID); c.Next() })
	r.POST("/api/v1/moderator/classes", h.CreateModeratorClass)

	body := `{"name":"Grade 8 B","grade":"8","section":"B"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/moderator/classes", bytes.NewBufferString(body))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %v: %s", w.Code, w.Body.String())
	}
	var class domain.Class
	if err := json.Unmarshal(w.Body.Bytes(), &class); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(class.InviteCode) != 6 {
		t.Fatalf("expected a 6-char invite code, got %q", class.InviteCode)
	}
	if class.TeacherID != teacher.ID {
		t.Fatalf("class must be owned by the caller, got teacher_id=%q", class.TeacherID)
	}
	t.Cleanup(func() { database.DB.Where("id = ?", class.ID).Delete(&domain.Class{}) })
}

// TestJoinClassByCode proves a student can join with the invite code and
// that an unknown code is an honest 404 (never a silent no-op).
func TestJoinClassByCode(t *testing.T) {
	h := newSchoolTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)
	class := newClassWithTeacher(t, teacher.ID)
	enrollCleanup(t, class.ID, student.ID)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", student.ID); c.Next() })
	r.POST("/api/v1/classes/join", h.JoinClass)

	// Valid code → 200 with the class
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/classes/join", bytes.NewBufferString(`{"code":"`+class.InviteCode+`"}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %v: %s", w.Code, w.Body.String())
	}
	var res struct {
		ClassID string `json:"class_id"`
		Name    string `json:"name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res.ClassID != class.ID {
		t.Fatalf("joined wrong class: %s", res.ClassID)
	}

	// Membership is real: the class roster contains the student.
	members, err := repository.NewSchoolRepository(database.DB).ClassMembers(context.Background(), class.ID)
	if err != nil {
		t.Fatalf("roster: %v", err)
	}
	found := false
	for _, m := range members {
		if m.ID == student.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("student not enrolled after join")
	}

	// Unknown code → 404 with a clear message
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/classes/join", bytes.NewBufferString(`{"code":"ZZZZZZ"}`))
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown code, got %v: %s", w2.Code, w2.Body.String())
	}
}

// TestImportRosterCSV proves the honest import contract: created students
// get one-time passwords, existing students are enrolled, invalid rows carry
// per-row reasons, and a non-owner teacher is refused (403).
func TestImportRosterCSV(t *testing.T) {
	h := newSchoolTestHandler(t)
	owner := newTestUser(t, domain.RoleModerator)
	other := newTestUser(t, domain.RoleModerator)
	class := newClassWithTeacher(t, owner.ID)

	// New + existing students: one valid, one bad email, one without name.
	existing := newTestUser(t, domain.RoleStudent)
	csv := "name,email,phone,password\n" +
		"Sita Sharma,sita@test.local,9800000001,Pass@123\n" +
		"Bad Row,not-an-email,,\n" +
		",noname@test.local,,\n" +
		existing.Name + "," + existing.Email + ",,"

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("file", "roster.csv")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if _, err := fw.Write([]byte(csv)); err != nil {
		t.Fatalf("write form: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", owner.ID); c.Next() })
	r.POST("/api/v1/moderator/classes/:id/roster/import", h.ImportClassRoster)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/moderator/classes/"+class.ID+"/roster/import", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %v: %s", w.Code, w.Body.String())
	}

	var report service.RosterImportReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if report.Imported != 2 {
		t.Fatalf("expected 2 imported (1 new + 1 existing), got %d: %+v", report.Imported, report)
	}
	// The new student's one-time password is returned exactly once.
	pwd, ok := report.Passwords["sita@test.local"]
	if !ok || len(pwd) == 0 {
		t.Fatalf("expected a one-time password for the new student, got %+v", report.Passwords)
	}
	if len(report.Errors) != 2 {
		t.Fatalf("expected 2 per-row errors, got %+v", report.Errors)
	}
	for _, e := range report.Errors {
		if e.Row < 2 || e.Reason == "" {
			t.Fatalf("error rows must be 1-based file lines with reasons: %+v", e)
		}
	}

	// The generated password actually works for login (no fabricated state).
	user, err := repository.NewUserRepository(database.DB).FindByEmail(context.Background(), "sita@test.local")
	if err != nil {
		t.Fatalf("find imported user: %v", err)
	}
	if !service.CheckPasswordHash(pwd, user.PasswordHash) {
		t.Fatal("imported password does not verify")
	}
	t.Cleanup(func() {
		database.DB.Unscoped().Where("id = ?", user.ID).Delete(&domain.User{})
		database.DB.Where("email = ?", "sita@test.local").Unscoped().Delete(&domain.User{})
	})

	// Non-owner teacher → 403
	body2 := &bytes.Buffer{}
	mw2 := multipart.NewWriter(body2)
	fw2, err := mw2.CreateFormFile("file", "roster.csv")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if _, err := fw2.Write([]byte(csv)); err != nil {
		t.Fatalf("write form: %v", err)
	}
	if err := mw2.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("userID", other.ID); c.Next() })
	r2.POST("/api/v1/moderator/classes/:id/roster/import", h.ImportClassRoster)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/moderator/classes/"+class.ID+"/roster/import", body2)
	req2.Header.Set("Content-Type", mw2.FormDataContentType())
	r2.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner import, got %v: %s", w2.Code, w2.Body.String())
	}
}

// TestModeratorStudentProgressScoped proves the WP-1.1 status engine feeds
// the per-student view and the scope gate is a hard 404 for other teachers.
func TestModeratorStudentProgressScoped(t *testing.T) {
	h := newModeratorProgressHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	other := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)
	class := newClassWithTeacher(t, teacher.ID)

	// Enroll the student in the teacher's class.
	if err := repository.NewSchoolRepository(database.DB).Enroll(context.Background(), class.ID, []string{student.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	enrollCleanup(t, class.ID, student.ID)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", teacher.ID); c.Next() })
	r.GET("/api/v1/moderator/students/:id", h.GetModeratorStudentProgress)

	// Owner teacher sees the student with the canonical status vocabulary.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/moderator/students/"+student.ID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner, got %v: %s", w.Code, w.Body.String())
	}
	var payload struct {
		Activities []map[string]interface{} `json:"activities"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	statuses := make(map[string]bool)
	for _, a := range payload.Activities {
		statuses[a["status"].(string)] = true
	}
	if len(statuses) == 0 {
		t.Fatal("expected per-activity canonical statuses")
	}
	// Only canonical statuses may ever appear (WP-1.1 vocabulary).
	for st := range statuses {
		switch st {
		case domain.StatusNotStarted, domain.StatusActive, domain.StatusNeedsPractice, domain.StatusCompleted:
		default:
			t.Fatalf("non-canonical status leaked to teacher view: %q", st)
		}
	}

	// Another teacher → hard 404, never a glimpse.
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("userID", other.ID); c.Next() })
	r2.GET("/api/v1/moderator/students/:id", h.GetModeratorStudentProgress)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/moderator/students/"+student.ID, nil)
	r2.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner teacher, got %v: %s", w2.Code, w2.Body.String())
	}
}

// TestListMyClassesExposesInviteCode keeps the moderator class list usable
// for the wizard's "share this code" step.
func TestListMyClassesExposesInviteCode(t *testing.T) {
	h := newSchoolTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	class := newClassWithTeacher(t, teacher.ID)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", teacher.ID); c.Next() })
	r.GET("/api/v1/moderator/classes", h.ListMyClasses)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/moderator/classes", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %v", w.Code)
	}
	if !strings.Contains(w.Body.String(), class.InviteCode) {
		t.Fatalf("class list must expose invite_code: %s", w.Body.String())
	}
}
