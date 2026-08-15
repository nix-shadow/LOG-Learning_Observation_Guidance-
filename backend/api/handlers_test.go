package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/handler"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	// Setup
	gin.SetMode(gin.TestMode)
	
	// Change to root backend directory so log.db is created there
	os.Chdir("..")
	database.InitDB() // AutoMigrates and AutoSeeds

	// Run tests
	code := m.Run()

	// Exit
	os.Exit(code)
}

func setupTestRouter() *gin.Engine {
	r := gin.Default()
	return r
}

func TestGetCourses(t *testing.T) {
	r := setupTestRouter()
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(
		repository.NewModeratorRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewActivityRepository(database.DB),
	)
	learnerHandler := handler.NewLearnerHandler(learnerService, courseService, moderatorService)

	r.GET("/api/courses", learnerHandler.GetCourses)

	req, _ := http.NewRequest("GET", "/api/courses?page=1&limit=2", nil)
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

	courses, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data array in response")
	}

	if len(courses) != 2 {
		t.Fatalf("Expected 2 courses (due to limit=2), got %v", len(courses))
	}
}

func TestGetModeratorRoster(t *testing.T) {
	r := setupTestRouter()
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(
		repository.NewModeratorRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewActivityRepository(database.DB),
	)
	learnerHandler := handler.NewLearnerHandler(learnerService, courseService, moderatorService)

	r.GET("/api/moderator/roster", learnerHandler.GetModeratorRoster)

	req, _ := http.NewRequest("GET", "/api/moderator/roster", nil)
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
	database.DB.Create(&domain.User{ID: learnerID, Phone: "99000000", Role: domain.RoleStudent, IsVerified: true})
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
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(
		repository.NewModeratorRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewActivityRepository(database.DB),
	)
	learnerHandler := handler.NewLearnerHandler(learnerService, courseService, moderatorService)

	r.POST("/api/activities/:id/complete", learnerHandler.CompleteActivity)

	req, _ := http.NewRequest("POST", "/api/activities/act-1/complete", nil)
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
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(database.DB),
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(database.DB))
	moderatorService := service.NewModeratorService(
		repository.NewModeratorRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewActivityRepository(database.DB),
	)
	learnerHandler := handler.NewLearnerHandler(learnerService, courseService, moderatorService)

	r.GET("/api/moderator/roster", learnerHandler.GetModeratorRoster)

	req, _ := http.NewRequest("GET", "/api/moderator/roster", nil)
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

	// Cross-check: count students with a zero streak directly from the DB
	var expected int64
	database.DB.Model(&domain.Progress{}).
		Joins("JOIN users ON users.id = progresses.learner_id AND users.role = ?", domain.RoleStudent).
		Where("progresses.current_streak = 0").Count(&expected)
	if int64(needsAttention) != expected {
		t.Fatalf("Expected needs_attention=%v from DB, got %v", expected, int64(needsAttention))
	}
}

func TestSyncBulkRejectsInvalidPayload(t *testing.T) {
	r := setupTestRouter()
	syncHandler := handler.NewSyncHandler(service.NewSyncService(repository.NewSyncRepository(database.DB)))
	r.POST("/api/sync/bulk", syncHandler.SyncBulk)

	body := bytes.NewBufferString(`{"version":"1.0","data":"not-an-array"}`)
	req, _ := http.NewRequest("POST", "/api/sync/bulk", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for malformed sync payload, got %v", w.Code)
	}
}
