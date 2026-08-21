package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Point the database at a throwaway temp file so tests never touch the
	// real backend/data/log.db (which would rewrite seeded demo data).
	tmpDir, err := os.MkdirTemp("", "log-backend-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	_ = os.Setenv("DB_PATH", filepath.Join(tmpDir, "test.db"))
	database.InitDB() // AutoMigrates and AutoSeeds into the temp DB

	// Run tests
	code := m.Run()

	_ = os.Unsetenv("DB_PATH")
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

func setupTestRouter() *gin.Engine {
	r := gin.Default()
	return r
}

func TestGetCourses(t *testing.T) {
	r := setupTestRouter()
	r.Use(func(c *gin.Context) { c.Set("userID", "user-123"); c.Next() })
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
		repository.NewCompletionRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(repository.NewModeratorRepository(database.DB))
	learnerHandler := NewLearnerHandler(learnerService, courseService, moderatorService, nil)

	r.GET("/api/v1/courses", learnerHandler.GetCourses)

	req, _ := http.NewRequest("GET", "/api/v1/courses?page=1&limit=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	courses, ok := response["courses"].([]interface{})
	if !ok {
		t.Fatalf("Expected courses array in response")
	}

	if len(courses) != 2 {
		t.Fatalf("Expected 2 courses (due to limit=2), got %v", len(courses))
	}

	// WP-0.2 C5: seeded enrollments (user-123 in course-1 and course-3) must
	// be visible per-learner, and every count must derive from real rows —
	// the old static seed numbers (1250, 980, …) must never surface.
	first, _ := courses[0].(map[string]interface{})
	enrolledNum, _ := first["enrolled"].(float64)
	enrolledFlag, _ := first["is_enrolled"].(bool)
	if enrolledFlag != (first["id"] == "course-1") {
		t.Fatalf("expected is_enrolled to reflect real enrollment, got course %v flag %v", first["id"], enrolledFlag)
	}
	if enrolledNum > 3 {
		t.Fatalf("enrolled count must be derived from real rows, got %v (legacy seed leaked?)", enrolledNum)
	}
}

// WP-0.2 C5: enrollment is persisted server-side, idempotent, and visible in
// the catalog immediately — no client-side-only state.
func TestEnrollUnenroll(t *testing.T) {
	r := setupTestRouter()
	r.Use(func(c *gin.Context) { c.Set("userID", "user-123"); c.Next() })
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
		repository.NewCompletionRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(repository.NewModeratorRepository(database.DB))
	learnerHandler := NewLearnerHandler(learnerService, courseService, moderatorService, nil)

	r.POST("/api/v1/courses/:id/enroll", learnerHandler.Enroll)
	r.DELETE("/api/v1/courses/:id/enroll", learnerHandler.Unenroll)
	r.GET("/api/v1/courses", learnerHandler.GetCourses)

	// Enroll in course-2 (not seeded for user-123).
	req, _ := http.NewRequest("POST", "/api/v1/courses/course-2/enroll", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on enroll, got %d", w.Code)
	}

	// Enrolling twice must not duplicate the row.
	req2, _ := http.NewRequest("POST", "/api/v1/courses/course-2/enroll", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200 on re-enroll, got %d", w2.Code)
	}
	var enrCount int64
	database.DB.Model(&domain.Enrollment{}).Where("user_id = ? AND course_id = ?", "user-123", "course-2").Count(&enrCount)
	if enrCount != 1 {
		t.Fatalf("Expected exactly 1 enrollment row after double-enroll, got %d", enrCount)
	}

	// A missing course must 404, never silently succeed.
	req3, _ := http.NewRequest("POST", "/api/v1/courses/nope/enroll", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 for unknown course, got %d", w3.Code)
	}

	// Catalog reflects the new enrollment with a real count.
	req4, _ := http.NewRequest("GET", "/api/v1/courses?page=1&limit=100", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	var resp struct {
		Courses []struct {
			ID         string `json:"id"`
			Enrolled   int    `json:"enrolled"`
			IsEnrolled bool   `json:"is_enrolled"`
		} `json:"courses"`
	}
	if err := json.Unmarshal(w4.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, c := range resp.Courses {
		if c.ID == "course-2" && !c.IsEnrolled {
			t.Fatalf("course-2 should be enrolled after POST enroll")
		}
		if c.ID == "course-2" && c.Enrolled != 1 {
			t.Fatalf("expected real count 1 for course-2, got %d", c.Enrolled)
		}
	}

	// Unenroll is idempotent and reflected immediately.
	req5, _ := http.NewRequest("DELETE", "/api/v1/courses/course-2/enroll", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Fatalf("Expected 200 on unenroll, got %d", w5.Code)
	}
	req6, _ := http.NewRequest("GET", "/api/v1/courses?page=1&limit=100", nil)
	w6 := httptest.NewRecorder()
	r.ServeHTTP(w6, req6)
	var resp2 struct {
		Courses []struct {
			ID         string `json:"id"`
			IsEnrolled bool   `json:"is_enrolled"`
		} `json:"courses"`
	}
	_ = json.Unmarshal(w6.Body.Bytes(), &resp2)
	for _, c := range resp2.Courses {
		if c.ID == "course-2" && c.IsEnrolled {
			t.Fatalf("course-2 should be unenrolled after DELETE")
		}
	}
}

func TestGetModeratorRoster(t *testing.T) {
	r := setupTestRouter()
	// Real identity: the roster is scoped to the calling teacher's classes.
	// (The old global-roster fallback that served without auth was removed.)
	r.Use(func(c *gin.Context) {
		c.Set("userID", "mod-1")
		c.Next()
	})
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
		repository.NewCompletionRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(repository.NewModeratorRepository(database.DB))
	learnerHandler := NewLearnerHandler(learnerService, courseService, moderatorService, nil)

	r.GET("/api/v1/moderator/roster", learnerHandler.GetModeratorRoster)

	req, _ := http.NewRequest("GET", "/api/v1/moderator/roster", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	roster, ok := response["roster"].([]interface{})
	if !ok {
		t.Fatalf("Expected roster array in response")
	}

	if len(roster) == 0 {
		t.Fatalf("Expected at least 1 student in the roster, got 0")
	}
}

func TestCompleteActivityCreatesProgressForNewLearner(t *testing.T) {
	learnerID := service.GenerateSecureID("user")
	phone := "99000000"
	database.DB.Create(&domain.User{ID: learnerID, Phone: &phone, Role: domain.RoleStudent, IsVerified: true})
	t.Cleanup(func() {
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Observation{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Guidance{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Progress{})
		database.DB.Where("id = ?", learnerID).Delete(&domain.User{})
	})

	r := setupTestRouter()
	// Stub the AuthMiddleware: derive the learner ID from a test header
	r.Use(func(c *gin.Context) {
		c.Set("userID", c.GetHeader("X-Test-Learner"))
		c.Next()
	})
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
		repository.NewCompletionRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(repository.NewModeratorRepository(database.DB))
	learnerHandler := NewLearnerHandler(learnerService, courseService, moderatorService, nil)

	r.POST("/api/v1/activities/:id/complete", learnerHandler.CompleteActivity)

	req, _ := http.NewRequest("POST", "/api/v1/activities/act-1/complete", nil)
	req.Header.Set("X-Test-Learner", learnerID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for new learner completion, got %v: %s", w.Code, w.Body.String())
	}

	var progress domain.Progress
	if err := database.DB.First(&progress, "learner_id = ?", learnerID).Error; err != nil {
		t.Fatalf("Expected progress record to be created for new learner: %v", err)
	}
	if progress.Completed != 1 {
		t.Fatalf("Expected completed=1 after first completion, got %v", progress.Completed)
	}
}

func TestGetModeratorRosterComputesNeedsAttention(t *testing.T) {
	r := setupTestRouter()
	r.Use(func(c *gin.Context) {
		c.Set("userID", "mod-1")
		c.Next()
	})
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
		repository.NewCompletionRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(repository.NewModeratorRepository(database.DB))
	learnerHandler := NewLearnerHandler(learnerService, courseService, moderatorService, nil)

	r.GET("/api/v1/moderator/roster", learnerHandler.GetModeratorRoster)

	req, _ := http.NewRequest("GET", "/api/v1/moderator/roster", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	needsAttention, ok := response["needs_attention"].(float64)
	if !ok {
		t.Fatalf("Expected computed needs_attention counter in response")
	}

	// Cross-check: count zero-streak students in THIS teacher's classes only
	var expected int64
	database.DB.Model(&domain.Progress{}).
		Joins("JOIN class_members ON class_members.user_id = progresses.learner_id").
		Joins("JOIN classes ON classes.id = class_members.class_id AND classes.teacher_id = ?", "mod-1").
		Where("progresses.current_streak = 0").Count(&expected)
	if int64(needsAttention) != expected {
		t.Fatalf("Expected needs_attention=%v from DB, got %v", expected, int64(needsAttention))
	}
}

func TestSyncBulkRejectsInvalidPayload(t *testing.T) {
	r := setupTestRouter()
	syncHandler := NewSyncHandler(service.NewSyncService(repository.NewSyncRepository(database.DB)))
	r.POST("/api/v1/sync/bulk", syncHandler.SyncBulk)

	body := bytes.NewBufferString(`{"version":"1.0","data":"not-an-array"}`)
	req, _ := http.NewRequest("POST", "/api/v1/sync/bulk", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for malformed sync payload, got %v", w.Code)
	}
}
