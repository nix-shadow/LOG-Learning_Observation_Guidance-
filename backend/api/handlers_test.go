package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"log-backend/database"
	"log-backend/models"

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
	r.GET("/api/courses", GetCourses)

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

	courses, ok := response["courses"].([]interface{})
	if !ok {
		t.Fatalf("Expected courses array in response")
	}

	if len(courses) != 2 {
		t.Fatalf("Expected 2 courses (due to limit=2), got %v", len(courses))
	}
}

func TestGetModeratorRoster(t *testing.T) {
	r := setupTestRouter()
	r.GET("/api/moderator/roster", GetModeratorRoster)

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
	learnerID := GenerateSecureID("user")
	database.DB.Create(&models.User{ID: learnerID, Phone: "99000000", Role: models.RoleStudent, IsVerified: true})
	t.Cleanup(func() {
		database.DB.Where("learner_id = ?", learnerID).Delete(&models.Observation{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&models.Guidance{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&models.Progress{})
		database.DB.Where("id = ?", learnerID).Delete(&models.User{})
	})

	r := setupTestRouter()
	// Stub the AuthMiddleware: derive the learner ID from a test header
	r.Use(func(c *gin.Context) {
		c.Set("userID", c.GetHeader("X-Test-Learner"))
		c.Next()
	})
	r.POST("/api/activities/:id/complete", CompleteActivity)

	req, _ := http.NewRequest("POST", "/api/activities/act-1/complete", nil)
	req.Header.Set("X-Test-Learner", learnerID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for new learner completion, got %v: %s", w.Code, w.Body.String())
	}

	var progress models.Progress
	if err := database.DB.First(&progress, "learner_id = ?", learnerID).Error; err != nil {
		t.Fatalf("Expected progress record to be created for new learner: %v", err)
	}
	if progress.Completed != 1 {
		t.Fatalf("Expected completed=1 after first completion, got %v", progress.Completed)
	}
}

func TestGetModeratorRosterComputesNeedsAttention(t *testing.T) {
	r := setupTestRouter()
	r.GET("/api/moderator/roster", GetModeratorRoster)

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
	database.DB.Model(&models.Progress{}).
		Joins("JOIN users ON users.id = progress.learner_id AND users.role = ?", models.RoleStudent).
		Where("progress.current_streak = 0").Count(&expected)
	if int64(needsAttention) != expected {
		t.Fatalf("Expected needs_attention=%v from DB, got %v", expected, int64(needsAttention))
	}
}

func TestSyncBulkRejectsInvalidPayload(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/sync/bulk", SyncBulk)

	body := bytes.NewBufferString(`{"version":"1.0","data":"not-an-array"}`)
	req, _ := http.NewRequest("POST", "/api/sync/bulk", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for malformed sync payload, got %v", w.Code)
	}
}
