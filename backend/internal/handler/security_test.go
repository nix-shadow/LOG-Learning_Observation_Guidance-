package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// TestAuthMiddlewareRejectsDemotedUser proves that a token issued for a role
// the user no longer holds is rejected: the middleware re-loads the user from
// the DB and compares roles instead of trusting the claims.
func TestAuthMiddlewareRejectsDemotedUser(t *testing.T) {
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)

	user := &domain.User{
		ID:         service.GenerateSecureID("user"),
		Email:      "demote@test.local",
		Name:       "Demote Me",
		Role:       domain.RoleAdmin,
		IsVerified: true,
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Unscoped().Where("id = ?", user.ID).Delete(&domain.User{})
	})

	// Token minted while the user was still an ADMIN.
	token, err := service.GenerateJWT(user.ID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to mint token: %v", err)
	}

	r := gin.New()
	r.GET("/api/v1/admin/dashboard", AuthMiddleware(authRepo, userRepo, repository.NewSchoolRepository(database.DB), domain.RoleAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Demote AFTER the token was issued.
	user.Role = domain.RoleStudent
	if err := userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("failed to demote user: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 for demoted user, got %v: %s", w.Code, w.Body.String())
	}
}

// TestAuthMiddlewareRejectsDeletedUser proves a soft-deleted account loses
// access immediately, even with a valid unexpired token.
func TestAuthMiddlewareRejectsDeletedUser(t *testing.T) {
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)

	user := &domain.User{
		ID:         service.GenerateSecureID("user"),
		Email:      "deleted@test.local",
		Name:       "Delete Me",
		Role:       domain.RoleStudent,
		IsVerified: true,
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token, err := service.GenerateJWT(user.ID, domain.RoleStudent)
	if err != nil {
		t.Fatalf("failed to mint token: %v", err)
	}

	database.DB.Delete(&domain.User{}, "id = ?", user.ID)

	r := gin.New()
	r.GET("/api/v1/dashboard", AuthMiddleware(authRepo, userRepo, repository.NewSchoolRepository(database.DB), domain.RoleStudent), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 for deleted user, got %v: %s", w.Code, w.Body.String())
	}
}

// TestSyncBulkIdempotent proves replaying an already-synced completion does
// not double-count progress, streak, score, observations, or daily activity.
func TestSyncBulkIdempotent(t *testing.T) {
	learnerID := service.GenerateSecureID("user")
	phone := "99000011"
	database.DB.Create(&domain.User{ID: learnerID, Phone: &phone, Role: domain.RoleStudent, IsVerified: true})
	t.Cleanup(func() {
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Observation{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Guidance{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Progress{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.LearnerActivity{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.DailyActivity{})
		database.DB.Where("id = ?", learnerID).Delete(&domain.User{})
	})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", learnerID)
		c.Next()
	})
	syncHandler := NewSyncHandler(service.NewSyncService(repository.NewSyncRepository(database.DB)))
	r.POST("/api/v1/sync/bulk", syncHandler.SyncBulk)

	payload := `{"version":"1.0","data":[{"endpoint":"/activities/act-1/complete","method":"POST"}]}`

	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/sync/bulk", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("sync attempt %d failed: %v %s", i+1, w.Code, w.Body.String())
		}
	}

	var progress domain.Progress
	if err := database.DB.First(&progress, "learner_id = ?", learnerID).Error; err != nil {
		t.Fatalf("expected progress record: %v", err)
	}
	if progress.Completed != 1 {
		t.Fatalf("expected completed=1 after idempotent replay, got %v", progress.Completed)
	}
	if progress.CurrentStreak != 1 {
		t.Fatalf("expected streak=1 after idempotent replay, got %v", progress.CurrentStreak)
	}
	if progress.OverallScore != 2.5 {
		t.Fatalf("expected score=2.5 after idempotent replay, got %v", progress.OverallScore)
	}

	var obsCount int64
	database.DB.Model(&domain.Observation{}).Where("learner_id = ?", learnerID).Count(&obsCount)
	if obsCount != 1 {
		t.Fatalf("expected exactly 1 observation after replay, got %v", obsCount)
	}

	var daily domain.DailyActivity
	if err := database.DB.First(&daily, "learner_id = ?", learnerID).Error; err != nil {
		t.Fatalf("expected daily activity row for synced completion: %v", err)
	}
	if daily.Score != 100.0 {
		t.Fatalf("expected daily score=100 after replay, got %v", daily.Score)
	}
}

// TestSyncBulkImprovingReplayKeepsBestScore proves an improving re-attempt
// through the sync endpoint updates the attempt facts (best score, attempts)
// without double-bumping progress, streak, or daily score.
func TestSyncBulkImprovingReplayKeepsBestScore(t *testing.T) {
	learnerID := service.GenerateSecureID("user")
	phone := "99000022"
	database.DB.Create(&domain.User{ID: learnerID, Phone: &phone, Role: domain.RoleStudent, IsVerified: true})
	t.Cleanup(func() {
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Observation{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Guidance{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Progress{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.LearnerActivity{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.DailyActivity{})
		database.DB.Where("id = ?", learnerID).Delete(&domain.User{})
	})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", learnerID)
		c.Next()
	})
	syncHandler := NewSyncHandler(service.NewSyncService(repository.NewSyncRepository(database.DB)))
	r.POST("/api/v1/sync/bulk", syncHandler.SyncBulk)

	first := `{"version":"1.0","data":[{"endpoint":"/activities/act-1/complete","method":"POST","body":"{\"correct_count\":5,\"total_count\":10,\"elapsed_seconds\":60}"}]}`
	improved := `{"version":"1.0","data":[{"endpoint":"/activities/act-1/complete","method":"POST","body":"{\"correct_count\":9,\"total_count\":10,\"elapsed_seconds\":90}"}]}`

	for i, payload := range []string{first, improved} {
		req, _ := http.NewRequest("POST", "/api/v1/sync/bulk", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("sync attempt %d failed: %v %s", i+1, w.Code, w.Body.String())
		}
	}

	// Progress was bumped exactly once (no double count on improvement)
	var progress domain.Progress
	if err := database.DB.First(&progress, "learner_id = ?", learnerID).Error; err != nil {
		t.Fatalf("expected progress record: %v", err)
	}
	if progress.Completed != 1 {
		t.Fatalf("expected completed=1 after improving replay, got %v", progress.Completed)
	}
	if progress.CurrentStreak != 1 {
		t.Fatalf("expected streak=1 after improving replay, got %v", progress.CurrentStreak)
	}

	// Daily activity reflects one completion, not two
	var daily domain.DailyActivity
	if err := database.DB.First(&daily, "learner_id = ?", learnerID).Error; err != nil {
		t.Fatalf("expected daily activity row: %v", err)
	}
	if daily.Score != 100.0 {
		t.Fatalf("expected daily score=100 (single completion), got %v", daily.Score)
	}
	if daily.Duration != 60 {
		t.Fatalf("expected duration from first completion (improving replay does not re-accumulate daily rows), got %v", daily.Duration)
	}

	// Attempt facts keep the best accuracy and count the improvement
	var la domain.LearnerActivity
	if err := database.DB.First(&la, "learner_id = ? AND activity_id = ?", learnerID, "act-1").Error; err != nil {
		t.Fatalf("expected learner activity row: %v", err)
	}
	if la.Attempts != 2 {
		t.Fatalf("expected attempts=2 after improvement, got %v", la.Attempts)
	}
	if la.Accuracy != 0.9 {
		t.Fatalf("expected accuracy=0.9 (best kept), got %v", la.Accuracy)
	}

	// New supportive guidance generated for the improvement; pure replay
	// (third send, same stats) must NOT create another one.
	var guiCount int64
	database.DB.Model(&domain.Guidance{}).Where("learner_id = ?", learnerID).Count(&guiCount)
	if guiCount != 2 {
		t.Fatalf("expected 2 guidance rows (first + improvement), got %v", guiCount)
	}
	req, _ := http.NewRequest("POST", "/api/v1/sync/bulk", bytes.NewBufferString(improved))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("replay failed: %v %s", w.Code, w.Body.String())
	}
	database.DB.Model(&domain.Guidance{}).Where("learner_id = ?", learnerID).Count(&guiCount)
	if guiCount != 2 {
		t.Fatalf("expected still 2 guidance rows after equal replay, got %v", guiCount)
	}
}

// TestRequestOTPCooldownReturns429 proves re-requesting an OTP inside the
// cooldown window is a 429, not a misleading 500.
func TestRequestOTPCooldownReturns429(t *testing.T) {
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)
	authService := service.NewAuthService(userRepo, authRepo)
	schoolService := service.NewSchoolService(repository.NewSchoolRepository(database.DB))
	authHandler := NewAuthHandler(authService, schoolService)

	phone := "9800000042"
	ctx := context.Background()
	t.Cleanup(func() {
		authRepo.DeleteOTP(ctx, phone)
		database.DB.Where("phone = ?", phone).Delete(&domain.User{})
	})

	r := gin.New()
	r.POST("/api/v1/auth/request-otp", authHandler.RequestOTP)

	body := `{"phone":"` + phone + `"}`
	for i := 1; i <= 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/request-otp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if i == 1 {
			if w.Code != http.StatusOK {
				t.Fatalf("first request: expected 200, got %v: %s", w.Code, w.Body.String())
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Fatalf("cooldown request: expected 429, got %v: %s", w.Code, w.Body.String())
			}
		}
	}

	// The live OTP must survive the rejected re-request
	record, err := authRepo.FindOTPByPhone(ctx, phone)
	if err != nil || record == nil {
		t.Fatalf("expected live OTP record to survive cooldown rejection, got %v", record)
	}
}

// TestVerifyOTPPerPhoneLimit proves 5 failed verify attempts invalidate the
// OTP: the record is deleted, so even the correct code is then rejected.
func TestVerifyOTPPerPhoneLimit(t *testing.T) {
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)
	authService := service.NewAuthService(userRepo, authRepo)

	phone := "9800000009"
	ctx := context.Background()
	t.Cleanup(func() {
		authRepo.DeleteOTP(ctx, phone)
		database.DB.Where("phone = ?", phone).Delete(&domain.User{})
	})

	if err := authService.RequestOTP(ctx, phone); err != nil {
		t.Fatalf("failed to request OTP: %v", err)
	}

	for i := 1; i <= 5; i++ {
		_, _, err := authService.VerifyOTP(ctx, phone, "000000")
		if i < 5 {
			if err == nil || err.Error() != "Invalid OTP" {
				t.Fatalf("attempt %d: expected Invalid OTP, got %v", i, err)
			}
		} else {
			if err == nil || err.Error() != "Too many incorrect attempts. Please request a new OTP" {
				t.Fatalf("attempt %d: expected limit message, got %v", i, err)
			}
		}
	}

	record, err := authRepo.FindOTPByPhone(ctx, phone)
	if err == nil || record != nil {
		t.Fatalf("expected OTP record to be deleted after 5 failures, got %+v", record)
	}
}

// TestCompleteActivityWritesDailyActivity proves the chart-data source of
// truth is written on completion (no fabricated fallback needed).
func TestCompleteActivityWritesDailyActivity(t *testing.T) {
	learnerID := service.GenerateSecureID("user")
	phone := "99000022"
	database.DB.Create(&domain.User{ID: learnerID, Phone: &phone, Role: domain.RoleStudent, IsVerified: true})
	t.Cleanup(func() {
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Observation{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Guidance{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.Progress{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.LearnerActivity{})
		database.DB.Where("learner_id = ?", learnerID).Delete(&domain.DailyActivity{})
		database.DB.Where("id = ?", learnerID).Delete(&domain.User{})
	})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", learnerID)
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
	learnerHandler := NewLearnerHandler(learnerService, courseService, moderatorService)
	r.POST("/api/v1/activities/:id/complete", learnerHandler.CompleteActivity)

	req, _ := http.NewRequest("POST", "/api/v1/activities/act-2/complete", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %v: %s", w.Code, w.Body.String())
	}

	var daily domain.DailyActivity
	if err := database.DB.First(&daily, "learner_id = ?", learnerID).Error; err != nil {
		t.Fatalf("expected daily activity row: %v", err)
	}
	if daily.Score != 100.0 {
		t.Fatalf("expected daily score=100, got %v", daily.Score)
	}
	if daily.DayName == "" {
		t.Fatal("expected a day name on the daily activity row")
	}
}

var _ = json.Marshal // keep encoding/json import if unused elsewhere

// TestGoogleAuthBareEmailRejected proves a bare {email} body (the old
// forgeable contract) is rejected by binding: no identity can be claimed
// without a real Google id_token.
func TestGoogleAuthBareEmailRejected(t *testing.T) {
	authService := service.NewAuthService(repository.NewUserRepository(database.DB), repository.NewAuthRepository(database.DB))
	schoolService := service.NewSchoolService(repository.NewSchoolRepository(database.DB))
	authHandler := NewAuthHandler(authService, schoolService)

	r := gin.New()
	r.POST("/api/v1/auth/google", authHandler.GoogleAuth)

	req, _ := http.NewRequest("POST", "/api/v1/auth/google", bytes.NewBufferString(`{"email":"learner@gmail.com","name":"Google Learner"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for bare email body, got %v: %s", w.Code, w.Body.String())
	}
}

// TestGoogleAuthMissingConfigRejected proves the endpoint fails closed when
// GOOGLE_CLIENT_ID is not configured on the server.
func TestGoogleAuthMissingConfigRejected(t *testing.T) {
	authService := service.NewAuthService(repository.NewUserRepository(database.DB), repository.NewAuthRepository(database.DB))
	schoolService := service.NewSchoolService(repository.NewSchoolRepository(database.DB))
	authHandler := NewAuthHandler(authService, schoolService)

	r := gin.New()
	r.POST("/api/v1/auth/google", authHandler.GoogleAuth)

	previous, had := os.LookupEnv("GOOGLE_CLIENT_ID")
	os.Unsetenv("GOOGLE_CLIENT_ID")
	t.Cleanup(func() {
		if had {
			os.Setenv("GOOGLE_CLIENT_ID", previous)
		}
	})

	req, _ := http.NewRequest("POST", "/api/v1/auth/google", bytes.NewBufferString(`{"token":"eyJhbGciOiJub25lIn0.eyJlbWFpbCI6ImxlYXJuZXJAZ21haWwuY29tIn0."}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 when Google Auth is unconfigured, got %v: %s", w.Code, w.Body.String())
	}
}

// TestGoogleAuthRejectsForgedToken proves a self-signed / garbage token
// cannot mint a LOG session: idtoken.Validate rejects it server-side.
func TestGoogleAuthRejectsForgedToken(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id.apps.googleusercontent.com")

	authService := service.NewAuthService(repository.NewUserRepository(database.DB), repository.NewAuthRepository(database.DB))
	schoolService := service.NewSchoolService(repository.NewSchoolRepository(database.DB))
	authHandler := NewAuthHandler(authService, schoolService)

	r := gin.New()
	r.POST("/api/v1/auth/google", authHandler.GoogleAuth)

	// Hand-crafted JWT claiming a verified google email — no real Google signature.
	forged := `{"token":"eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6ImhhY2tlckBnbWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IkhhY2tlciJ9.aW52YWxpZA"}`
	req, _ := http.NewRequest("POST", "/api/v1/auth/google", bytes.NewBufferString(forged))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 for forged Google token, got %v: %s", w.Code, w.Body.String())
	}
}
