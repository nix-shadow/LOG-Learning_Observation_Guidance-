package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

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
	database.DB.Create(&domain.User{ID: learnerID, Name: "Sync Idempotence Tester", Email: learnerID + "@sync.test", Phone: &phone, Role: domain.RoleStudent, IsVerified: true})
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
	database.DB.Create(&domain.User{ID: learnerID, Name: "Sync Best-Score Tester", Email: learnerID + "@sync.test", Phone: &phone, Role: domain.RoleStudent, IsVerified: true})
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
		_ = authRepo.DeleteOTP(ctx, phone)
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
		_ = authRepo.DeleteOTP(ctx, phone)
		database.DB.Where("phone = ?", phone).Delete(&domain.User{})
	})

	if err := authService.RequestOTP(ctx, phone); err != nil {
		t.Fatalf("failed to request OTP: %v", err)
	}

	for i := 1; i <= 5; i++ {
		_, _, err := authService.VerifyOTP(ctx, phone, "000000")
		if i < 5 {
			if err == nil || err.Error() != "invalid OTP" {
				t.Fatalf("attempt %d: expected invalid OTP, got %v", i, err)
			}
		} else {
			if err == nil || err.Error() != "too many incorrect attempts, please request a new OTP" {
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
	phone := service.GenerateSecureID("ph")
	// Unique email: soft-deleted rows keep their unique-index entries, so a
	// shared empty string would block every later test run on the same DB.
	email := service.GenerateSecureID("em") + "@test.local"
	database.DB.Create(&domain.User{ID: learnerID, Email: email, Phone: &phone, Role: domain.RoleStudent, IsVerified: true})
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
	learnerHandler := NewLearnerHandler(learnerService, courseService, moderatorService, nil)
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
	// WP-1.2 RC-02 practice metrics: a quiz-less completion carries no
	// accuracy signal (honest 0), but it is still a real practice attempt.
	if daily.Attempts != 1 {
		t.Fatalf("expected attempts=1, got %v", daily.Attempts)
	}
	if daily.Accuracy != 0 {
		t.Fatalf("expected accuracy=0 for quiz-less completion, got %v", daily.Accuracy)
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
	_ = os.Unsetenv("GOOGLE_CLIENT_ID")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("GOOGLE_CLIENT_ID", previous)
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

// Regression: per-route rate-limit budgets used to be capped by the shared
// `rateLimitMax` constant (5) instead of the limiter's own `limit`, so a
// RateLimitLogin=10 bucket silently 429'd at the 5th request. Live-stack
// testing against the Docker image caught it (scripts/live_stack_test.py).
func TestRateLimiterHonorsPerRouteBudget(t *testing.T) {
	rl := newRateLimiter(RateLimitLogin, time.Minute) // budget 10/min
	for i := 1; i <= RateLimitLogin; i++ {
		ok, remaining := rl.allow("10.0.0.1")
		if !ok {
			t.Fatalf("request %d/%d: expected allowed (budget %d), got blocked (remaining=%d)",
				i, RateLimitLogin, RateLimitLogin, remaining)
		}
	}
	if ok, _ := rl.allow("10.0.0.1"); ok {
		t.Fatal("request beyond the budget must be blocked")
	}
	// A different IP starts fresh — the budget is per client, not global.
	if ok, _ := rl.allow("10.0.0.2"); !ok {
		t.Fatal("a new client must not inherit the first client's consumption")
	}
}

func TestRateLimiterWindowResets(t *testing.T) {
	rl := newRateLimiter(2, 60*time.Millisecond)
	if ok, _ := rl.allow("10.0.0.9"); !ok {
		t.Fatal("first request must be allowed")
	}
	if ok, _ := rl.allow("10.0.0.9"); !ok {
		t.Fatal("second request must be allowed")
	}
	if ok, _ := rl.allow("10.0.0.9"); ok {
		t.Fatal("third request within the window must be blocked")
	}
	time.Sleep(80 * time.Millisecond)
	if ok, _ := rl.allow("10.0.0.9"); !ok {
		t.Fatal("request after the window elapsed must be allowed again")
	}
}

// Registration (live-stack finding): the email/password register route was
// dropped in the backend rewrite while the frontend kept calling it —
// /auth/register answered 404 and the Register tab could never create an
// account. Ported through the service seam; these tests pin it.
func TestRegisterCreatesStudentAndReturnsToken(t *testing.T) {
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)
	authService := service.NewAuthService(userRepo, authRepo)
	authHandler := NewAuthHandler(authService, service.NewSchoolService(repository.NewSchoolRepository(database.DB)))

	email := "register-test@log.edu"
	t.Cleanup(func() {
		database.DB.Where("email = ?", email).Delete(&domain.User{})
	})

	r := gin.New()
	r.POST("/api/v1/auth/register", authHandler.Register)

	body := `{"name":"Register Test","email":"` + email + `","password":"Password123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %v: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
		User  struct {
			ID    string      `json:"id"`
			Role  domain.Role `json:"role"`
			Email string      `json:"email"`
			Phone *string     `json:"phone"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad response json: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("registration must return a usable token")
	}
	if resp.User.Role != domain.RoleStudent {
		t.Fatalf("new accounts must be STUDENT, got %v", resp.User.Role)
	}
	if resp.User.Phone != nil {
		t.Fatal("new email accounts must not invent a phone number")
	}
}

func TestRegisterDuplicateEmailReturns409(t *testing.T) {
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)
	authService := service.NewAuthService(userRepo, authRepo)
	authHandler := NewAuthHandler(authService, service.NewSchoolService(repository.NewSchoolRepository(database.DB)))

	r := gin.New()
	r.POST("/api/v1/auth/register", authHandler.Register)

	body := `{"name":"Dup","email":"aisha@example.com","password":"Password123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for existing email, got %v: %s", w.Code, w.Body.String())
	}
}

func TestRegisterInvalidPayloadReturns400(t *testing.T) {
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)
	authHandler := NewAuthHandler(service.NewAuthService(userRepo, authRepo), service.NewSchoolService(repository.NewSchoolRepository(database.DB)))

	r := gin.New()
	r.POST("/api/v1/auth/register", authHandler.Register)

	for _, body := range []string{`{"name":"X","email":"nope","password":"short"}`,
		`{"name":"","email":"a@b.co","password":"longenough"}`,
		`not json`} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %q: expected 400, got %v", body, w.Code)
		}
	}
}
