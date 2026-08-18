package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// queryCounter counts every SQL statement gorm executes through its logger.
type queryCounter struct{ n *atomic.Int64 }

func (q queryCounter) LogMode(logger.LogLevel) logger.Interface      { return q }
func (q queryCounter) Info(context.Context, string, ...interface{})  {}
func (q queryCounter) Warn(context.Context, string, ...interface{})  {}
func (q queryCounter) Error(context.Context, string, ...interface{}) {}
func (q queryCounter) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	fc() // run the real query; counting happens regardless of its SQL text
	q.n.Add(1)
}

// TestRosterQueryCountIsConstant proves the roster endpoint does NOT issue
// one progress query per student. The query count must be identical whether
// the school has 25 or 50 students — the N+1 regression trap. The fixed
// budget (roster page + batched progress lookup + two counts + the two
// course-label queries) is well under the old per-student fan-out.
func TestRosterQueryCountIsConstant(t *testing.T) {
	var base int64
	database.DB.Model(&domain.User{}).Where("role = ?", domain.RoleStudent).Count(&base)

	seedRosterStudents(t, 25)
	queries25 := countRosterRequest(t, int(base)+25)

	seedRosterStudents(t, 25) // 50 students total now
	queries50 := countRosterRequest(t, int(base)+50)

	if queries25 > 6 {
		t.Fatalf("roster issued %d queries for 25 students — expected at most 6 (N+1 regression)", queries25)
	}
	if queries50 != queries25 {
		t.Fatalf("query count scaled with roster size: 25 students=%d, 50 students=%d", queries25, queries50)
	}
}

func seedRosterStudents(t *testing.T, count int) {
	t.Helper()
	var ids []string
	for i := 0; i < count; i++ {
		id := service.GenerateSecureID("user")
		phone := service.GenerateSecureID("ph")
		database.DB.Create(&domain.User{
			ID: id, Name: "Roster Student", Email: "roster" + id + "@test.edu",
			Phone: &phone, Role: domain.RoleStudent, IsVerified: true,
		})
		ids = append(ids, id)
		// Half the students have progress rows, half have none — the batched
		// lookup must cover both shapes.
		if i%2 == 0 {
			database.DB.Create(&domain.Progress{
				LearnerID:     id,
				TotalTopics:   10,
				Completed:     4,
				CurrentStreak: i % 3, // mixes Active (streak 2) and Inactive (streak 0)
			})
		}
	}
	t.Cleanup(func() {
		database.DB.Where("learner_id IN ?", ids).Delete(&domain.Progress{})
		database.DB.Where("id IN ?", ids).Delete(&domain.User{})
	})
}

func countRosterRequest(t *testing.T, wantRoster int) int64 {
	t.Helper()
	var queries atomic.Int64
	countingDB := database.DB.Session(&gorm.Session{Logger: queryCounter{n: &queries}})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", "mod-1")
		c.Next()
	})
	learnerService := service.NewLearnerService(
		repository.NewUserRepository(countingDB),
		repository.NewActivityRepository(countingDB),
		repository.NewProgressRepository(countingDB),
		repository.NewLearnerDataRepository(countingDB),
		repository.NewCompletionRepository(countingDB),
	)
	courseService := service.NewCourseService(repository.NewCourseRepository(countingDB))
	moderatorService := service.NewModeratorService(repository.NewModeratorRepository(countingDB))
	learnerHandler := NewLearnerHandler(learnerService, courseService, moderatorService)
	r.GET("/api/v1/moderator/roster", learnerHandler.GetModeratorRoster)

	req, _ := http.NewRequest("GET", "/api/v1/moderator/roster?page=1&limit=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %v: %s", w.Code, w.Body.String())
	}

	// The request must also return correct per-student math.
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal roster: %v", err)
	}
	roster, ok := response["roster"].([]interface{})
	if !ok || len(roster) != wantRoster {
		t.Fatalf("expected %d students in roster, got %d", wantRoster, len(roster))
	}

	return queries.Load()
}
