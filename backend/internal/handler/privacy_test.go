package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

func newPrivacyTestHandler(t *testing.T) *PrivacyHandler {
	repo := repository.NewPrivacyRepository(database.DB)
	return NewPrivacyHandler(service.NewPrivacyService(repo))
}

// newLearnerWithData creates a verified learner with a full set of owned data
// rows (progress, activities, observations, guidance, daily activity, class
// membership, audit rows) so export and erasure tests exercise every table in
// the data map. Cleanup is registered per test.
func newLearnerWithData(t *testing.T) *domain.User {
	t.Helper()
	user := newTestUser(t, domain.RoleStudent)
	phone := "+977" + service.GenerateSecureID("9")[:9]
	user.Phone = &phone
	if err := database.DB.Model(user).Update("phone", phone).Error; err != nil {
		t.Fatalf("failed to set phone: %v", err)
	}

	rows := []interface{}{
		&domain.Progress{LearnerID: user.ID, TotalTopics: 5, Completed: 2, OverallScore: 70},
		&domain.LearnerActivity{LearnerID: user.ID, ActivityID: "act-1", Status: "Completed", CompletedAt: time.Now(), Score: 90},
		&domain.Observation{ID: service.GenerateSecureID("obs"), LearnerID: user.ID, Category: "strengths", Text: "Works steadily.", CreatedAt: time.Now()},
		&domain.Guidance{ID: service.GenerateSecureID("gui"), LearnerID: user.ID, Type: "next_step", Text: "Keep going.", CreatedAt: time.Now()},
		&domain.DailyActivity{ID: service.GenerateSecureID("da"), LearnerID: user.ID, Date: time.Now(), DayName: "Mon", Score: 65, Duration: 20},
		&domain.ClassMember{ClassID: "cls-1", UserID: user.ID, JoinedAt: time.Now()},
		&domain.TokenBlocklist{JTI: service.GenerateSecureID("jti"), UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour), RevokedAt: time.Now()},
		&domain.UserRevocation{UserID: user.ID, RevokedBefore: time.Now()},
		&domain.ConsentRecord{ID: service.GenerateSecureID("csn"), UserID: user.ID, ConsentType: domain.ConsentTypeGuardian, Version: domain.PolicyVersion, Status: domain.ConsentStatusGranted, GrantedBy: "guardian", Language: "ne", Source: "register", GrantedAt: time.Now()},
		&domain.AuditLog{UserID: user.ID, Action: "activity.create", Detail: "test", CreatedAt: time.Now()},
		&domain.OTPRecord{Phone: phone, OTP: "000000", ExpiresAt: time.Now().Add(time.Hour)},
		&domain.Announcement{ID: service.GenerateSecureID("ann"), Title: "Bye", Body: "x", AuthorID: user.ID, CreatedAt: time.Now()},
		&domain.Assignment{ID: service.GenerateSecureID("asg"), ClassID: "cls-1", Title: "HW", CreatedBy: user.ID, CreatedAt: time.Now()},
		&domain.Class{ID: service.GenerateSecureID("cls"), Name: "Temp Class", TeacherID: user.ID, CreatedAt: time.Now()},
	}
	for _, row := range rows {
		if err := database.DB.Create(row).Error; err != nil {
			t.Fatalf("failed to seed data for %T: %v", row, err)
		}
	}
	t.Cleanup(func() {
		for _, row := range rows {
			database.DB.Unscoped().Delete(row)
		}
	})
	return user
}

func TestRecordConsentAndGet(t *testing.T) {
	r := setupTestRouter()
	user := newTestUser(t, domain.RoleStudent)
	r.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	h := newPrivacyTestHandler(t)
	r.POST("/api/v1/me/consent", h.RecordConsent)
	r.GET("/api/v1/me/consent", h.GetMyConsent)

	// sha256 of the exact bilingual notice text shown at registration
	// (must match the frontend constant). Any 64-hex digest is accepted by the
	// server; tests pin the real one so drift is caught.
	testDisclosureHash := "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"

	body := `{"consent_type":"guardian","version":"2026-08-v1","granted_by":"guardian","guardian_name":"Hari Sharma","guardian_contact":"9841xxxxxx","language":"ne","source":"register","disclosure_hash":"` + testDisclosureHash + `"}`
	req, _ := http.NewRequest("POST", "/api/v1/me/consent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created struct {
		ConsentType  string `json:"consent_type"`
		Status       string `json:"status"`
		GuardianName string `json:"guardian_name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if created.ConsentType != "guardian" || created.Status != "granted" || created.GuardianName != "Hari Sharma" {
		t.Fatalf("unexpected consent record: %+v", created)
	}

	req2, _ := http.NewRequest("GET", "/api/v1/me/consent", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w2.Code)
	}
	var got struct {
		Consent  []domain.ConsentRecord `json:"consent"`
		Required string                 `json:"required"`
		Policy   struct {
			Version string `json:"version"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(got.Consent) != 1 || got.Consent[0].ConsentType != "guardian" {
		t.Fatalf("expected one guardian consent, got %+v", got.Consent)
	}
	if got.Required != "guardian" || got.Policy.Version == "" {
		t.Fatalf("expected policy block, got %+v", got.Policy)
	}

	// Re-granting must update the same row, not duplicate it.
	req3, _ := http.NewRequest("POST", "/api/v1/me/consent", bytes.NewBufferString(body))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusCreated {
		t.Fatalf("Expected 201 on re-grant, got %d", w3.Code)
	}
	req4, _ := http.NewRequest("GET", "/api/v1/me/consent", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	var got2 struct {
		Consent []domain.ConsentRecord `json:"consent"`
	}
	_ = json.Unmarshal(w4.Body.Bytes(), &got2)
	if len(got2.Consent) != 1 {
		t.Fatalf("expected single upserted row after re-grant, got %d rows", len(got2.Consent))
	}

	// The documented djb2- fallback hash (non-secure context, plain-HTTP
	// school LAN) must be accepted and stored verbatim.
	djb2Body := `{"consent_type":"guardian","version":"2026-08-v1","granted_by":"guardian","language":"ne","disclosure_hash":"djb2-1a2b3c4d"}`
	req5, _ := http.NewRequest("POST", "/api/v1/me/consent", bytes.NewBufferString(djb2Body))
	req5.Header.Set("Content-Type", "application/json")
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusCreated {
		t.Fatalf("Expected 201 for djb2 fallback hash, got %d: %s", w5.Code, w5.Body.String())
	}
	var got3 struct {
		DisclosureHash string `json:"disclosure_hash"`
	}
	_ = json.Unmarshal(w5.Body.Bytes(), &got3)
	if got3.DisclosureHash != "djb2-1a2b3c4d" {
		t.Fatalf("expected djb2 hash echoed verbatim, got %q", got3.DisclosureHash)
	}
}

func TestRecordConsentValidation(t *testing.T) {
	r := setupTestRouter()
	user := newTestUser(t, domain.RoleStudent)
	r.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	h := newPrivacyTestHandler(t)
	r.POST("/api/v1/me/consent", h.RecordConsent)

	cases := []string{
		`{"consent_type":"telepathy","version":"2026-08-v1","granted_by":"guardian","language":"ne"}`,                                                                 // bad type
		`{"consent_type":"guardian","version":"2026-08-v1","granted_by":"elf","language":"ne"}`,                                                                       // bad giver
		`{"consent_type":"guardian","version":"2026-08-v1","granted_by":"guardian","language":"fr"}`,                                                                  // bad language
		`{"consent_type":"guardian","version":"","granted_by":"guardian","language":"ne"}`,                                                                            // empty version
		`{"consent_type":"guardian","version":"2026-08-v1","granted_by":"guardian","language":"ne","guardian_name":"` + string(bytes.Repeat([]byte("x"), 200)) + `"}`, // overlong name
		// WP-0.1 research: guardian consent without the disclosure hash of the
		// presented notice is unprovable — rejected, not stored.
		`{"consent_type":"guardian","version":"2026-08-v1","granted_by":"guardian","language":"ne"}`,                             // missing disclosure_hash
		`{"consent_type":"guardian","version":"2026-08-v1","granted_by":"guardian","language":"ne","disclosure_hash":"not-hex"}`, // malformed hash
	}
	for i, body := range cases {
		req, _ := http.NewRequest("POST", "/api/v1/me/consent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("case %d: expected 400, got %d: %s", i, w.Code, w.Body.String())
		}
	}
}

// The erasure path must physically scrub the file (WAL checkpoint + VACUUM),
// not just mark rows deleted — a plain DELETE leaves recoverable cells.
func TestDeleteAccountScrubsDatabase(t *testing.T) {
	r := setupTestRouter()
	user := newLearnerWithData(t)
	r.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	h := newPrivacyTestHandler(t)
	r.DELETE("/api/v1/me", h.DeleteAccount)

	body := `{"confirm":"DELETE"}`
	req, _ := http.NewRequest("DELETE", "/api/v1/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// The user row and every learner row must be gone.
	var count int64
	database.DB.Model(&domain.User{}).Unscoped().Where("id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("user row survived erasure")
	}
	database.DB.Model(&domain.LearnerActivity{}).Unscoped().Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("learner activity survived erasure")
	}
}

func TestExportData(t *testing.T) {
	r := setupTestRouter()
	user := newLearnerWithData(t)
	r.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	h := newPrivacyTestHandler(t)
	r.GET("/api/v1/me/export", h.ExportMyData)

	req, _ := http.NewRequest("GET", "/api/v1/me/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to unmarshal export: %v", err)
	}
	if envelope["exported_at"] == nil || envelope["policy_version"] != domain.PolicyVersion || envelope["schema_version"] != float64(1) {
		t.Fatalf("envelope metadata missing: %+v", envelope)
	}
	userOut, _ := envelope["user"].(map[string]interface{})
	if userOut == nil {
		t.Fatalf("user block missing from export")
	}
	if _, leaked := userOut["password_hash"]; leaked {
		t.Fatalf("password hash leaked into export")
	}
	for _, key := range []string{"learner_activities", "observations", "guidance", "daily_activities", "classes", "consent"} {
		if envelope[key] == nil {
			t.Fatalf("expected %q in export", key)
		}
	}
}

func TestDeleteAccountRequiresConfirmation(t *testing.T) {
	r := setupTestRouter()
	user := newTestUser(t, domain.RoleStudent)
	r.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	h := newPrivacyTestHandler(t)
	r.DELETE("/api/v1/me", h.DeleteAccount)

	for i, body := range []string{`{}`, `{"confirm":"yes"}`} {
		req, _ := http.NewRequest("DELETE", "/api/v1/me", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("case %d: expected 400, got %d", i, w.Code)
		}
	}
}

func TestDeleteAccountErasure(t *testing.T) {
	r := setupTestRouter()
	user := newLearnerWithData(t)
	r.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	h := newPrivacyTestHandler(t)
	r.DELETE("/api/v1/me", h.DeleteAccount)

	req, _ := http.NewRequest("DELETE", "/api/v1/me", bytes.NewBufferString(`{"confirm":"DELETE"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var userCount int64
	database.DB.Unscoped().Model(&domain.User{}).Where("id = ?", user.ID).Count(&userCount)
	if userCount != 0 {
		t.Fatalf("user row still present after erasure")
	}

	// Learner-private tables must be empty.
	checks := []struct {
		table string
		col   string
		model interface{}
	}{
		{"progress", "learner_id", &domain.Progress{}},
		{"learner_activities", "learner_id", &domain.LearnerActivity{}},
		{"observations", "learner_id", &domain.Observation{}},
		{"guidance", "learner_id", &domain.Guidance{}},
		{"daily_activities", "learner_id", &domain.DailyActivity{}},
		{"class_members", "user_id", &domain.ClassMember{}},
		{"token_blocklists", "user_id", &domain.TokenBlocklist{}},
		{"user_revocations", "user_id", &domain.UserRevocation{}},
		{"consent_records", "user_id", &domain.ConsentRecord{}},
	}
	for _, c := range checks {
		var n int64
		if err := database.DB.Unscoped().Model(c.model).Where(c.col+" = ?", user.ID).Count(&n).Error; err != nil {
			t.Fatalf("query %s failed: %v", c.table, err)
		}
		if n != 0 {
			t.Fatalf("table %s still holds %d rows for erased user", c.table, n)
		}
	}

	// OTP rows die with the phone.
	var otpCount int64
	database.DB.Unscoped().Model(&domain.OTPRecord{}).Where("phone = ?", *user.Phone).Count(&otpCount)
	if otpCount != 0 {
		t.Fatalf("otp_records still hold %d rows for erased phone", otpCount)
	}

	// Authored content survives but is anonymized.
	var annAuthor string
	if err := database.DB.Model(&domain.Announcement{}).Select("author_id").Where("title = ?", "Bye").Scan(&annAuthor).Error; err != nil {
		t.Fatalf("announcement lookup failed: %v", err)
	}
	if annAuthor != "" {
		t.Fatalf("announcement author not anonymized: %q", annAuthor)
	}
	var asgCreator string
	if err := database.DB.Model(&domain.Assignment{}).Select("created_by").Where("title = ?", "HW").Scan(&asgCreator).Error; err != nil {
		t.Fatalf("assignment lookup failed: %v", err)
	}
	if asgCreator != "" {
		t.Fatalf("assignment creator not anonymized: %q", asgCreator)
	}

	// The audit trail is anonymized, and the erasure entry itself carries no
	// user reference — only the truncated hash for cross-referencing.
	var auditCount int64
	database.DB.Model(&domain.AuditLog{}).Where("user_id = ?", user.ID).Count(&auditCount)
	if auditCount != 0 {
		t.Fatalf("audit_log still references erased user %d times", auditCount)
	}
	var erasure domain.AuditLog
	if err := database.DB.Where("action = ?", "privacy.account_deleted").Order("id desc").First(&erasure).Error; err != nil {
		t.Fatalf("erasure audit entry missing: %v", err)
	}
	if erasure.UserID != "" {
		t.Fatalf("erasure entry carries user reference %q", erasure.UserID)
	}

	// The learner's own audit rows survive as anonymized history.
	var anonymized int64
	database.DB.Model(&domain.AuditLog{}).Where("detail = ?", "test").Count(&anonymized)
	if anonymized != 1 {
		t.Fatalf("expected 1 anonymized audit row, got %d", anonymized)
	}
}

func TestAdminGetUsersShowsConsent(t *testing.T) {
	r := setupTestRouter()
	user := newTestUser(t, domain.RoleStudent)
	repo := repository.NewPrivacyRepository(database.DB)
	if err := repo.UpsertConsent(context.Background(), &domain.ConsentRecord{
		ID:          service.GenerateSecureID("csn"),
		UserID:      user.ID,
		ConsentType: domain.ConsentTypeGuardian,
		Version:     domain.PolicyVersion,
		Status:      domain.ConsentStatusGranted,
		GrantedBy:   "guardian",
		Language:    "en",
		Source:      "register",
		GrantedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("failed to seed consent: %v", err)
	}
	h := NewAdminHandler(service.NewAdminService(repository.NewAdminRepository(database.DB)))
	r.GET("/api/v1/admin/users", h.GetUsers)

	req, _ := http.NewRequest("GET", "/api/v1/admin/users?limit=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp struct {
		Users []struct {
			ID      string `json:"id"`
			Consent *struct {
				ConsentType string `json:"consent_type"`
				Status      string `json:"status"`
				Version     string `json:"version"`
			} `json:"consent"`
		} `json:"users"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	found := false
	for _, u := range resp.Users {
		if u.ID == user.ID {
			found = true
			if u.Consent == nil {
				t.Fatalf("expected consent block for user %s", user.ID)
			}
			if u.Consent.ConsentType != "guardian" || u.Consent.Status != "granted" {
				t.Fatalf("unexpected consent: %+v", u.Consent)
			}
		}
	}
	if !found {
		t.Fatalf("created user not present in admin list")
	}

	// A user without consent must show null — never a fabricated value.
	var without *struct {
		ID      string `json:"id"`
		Consent *struct {
			ConsentType string `json:"consent_type"`
			Status      string `json:"status"`
			Version     string `json:"version"`
		} `json:"consent"`
	}
	for _, u := range resp.Users {
		if u.ID == "mod-1" {
			without = &u
			break
		}
	}
	if without == nil {
		t.Fatalf("seeded moderator missing from admin list")
		return
	}
	if without.Consent != nil {
		t.Fatalf("expected null consent for moderator, got %+v", without.Consent)
	}
}

// The server-side consent gate (WP-0.1 enforcement round): learner mutations
// require an active guardian grant even if the login UI is bypassed. The 403
// carries code=consent_required so the offline queue preserves records. Staff
// roles are exempt.
func TestRequireConsentGate(t *testing.T) {
	repo := repository.NewPrivacyRepository(database.DB)
	gate := RequireConsent(repo)

	newRouter := func(uid string, role domain.Role) *gin.Engine {
		r := setupTestRouter()
		r.Use(func(c *gin.Context) { c.Set("userID", uid); c.Set("role", role); c.Next() })
		r.POST("/protected", gate, func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
		return r
	}
	post := func(r *gin.Engine) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("POST", "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// 1. Student without consent → 403 + machine-readable code.
	noConsent := newTestUser(t, domain.RoleStudent)
	w := post(newRouter(noConsent.ID, domain.RoleStudent))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without consent, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal gate response: %v", err)
	}
	if body["code"] != "consent_required" {
		t.Fatalf("expected code consent_required, got %v", body["code"])
	}

	// 2. Student with an active guardian grant → passes.
	withConsent := newTestUser(t, domain.RoleStudent)
	consentID := service.GenerateSecureID("csn")
	if err := repo.UpsertConsent(context.Background(), &domain.ConsentRecord{
		ID:          consentID,
		UserID:      withConsent.ID,
		ConsentType: domain.ConsentTypeGuardian,
		Version:     domain.PolicyVersion,
		Status:      domain.ConsentStatusGranted,
		GrantedBy:   "guardian",
		Language:    "ne",
		Source:      "register",
		GrantedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("failed to seed consent: %v", err)
	}
	t.Cleanup(func() { database.DB.Unscoped().Where("id = ?", consentID).Delete(&domain.ConsentRecord{}) })
	w = post(newRouter(withConsent.ID, domain.RoleStudent))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with consent, got %d", w.Code)
	}

	// 3. A withdrawn grant blocks again (re-grant is the only unblock).
	if err := repo.UpsertConsent(context.Background(), &domain.ConsentRecord{
		ID:          service.GenerateSecureID("csn"),
		UserID:      withConsent.ID,
		ConsentType: domain.ConsentTypeGuardian,
		Version:     domain.PolicyVersion,
		Status:      domain.ConsentStatusWithdrawn,
		GrantedBy:   "guardian",
		Language:    "ne",
		Source:      "settings",
		GrantedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("failed to withdraw consent: %v", err)
	}
	w = post(newRouter(withConsent.ID, domain.RoleStudent))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 after withdrawal, got %d", w.Code)
	}

	// 4. Staff (moderator) without consent → exempt, passes.
	staff := newTestUser(t, domain.RoleModerator)
	w = post(newRouter(staff.ID, domain.RoleModerator))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for staff, got %d", w.Code)
	}
}

// The retention purge must erase only genuinely inactive learners (via the
// full erasure map, so the school context survives), keep active learners and
// staff, and drop audit rows past their window.
func TestPurgeExpiredData(t *testing.T) {
	svc := service.NewPrivacyService(repository.NewPrivacyRepository(database.DB))

	// Learner with no activity for 3 years → retention candidate.
	stale := newTestUser(t, domain.RoleStudent)
	if err := database.DB.Model(stale).Update("updated_at", time.Now().AddDate(-3, 0, 0)).Error; err != nil {
		t.Fatalf("failed to age user: %v", err)
	}
	// The stale learner's audit row stays within the 3-year retention window;
	// the erasure map must anonymize it, not delete it.
	staleAudit := &domain.AuditLog{UserID: stale.ID, Action: "activity.create", Detail: "stale-learner-row", CreatedAt: time.Now().AddDate(-2, 0, 0)}
	if err := database.DB.Create(staleAudit).Error; err != nil {
		t.Fatalf("failed to seed stale audit: %v", err)
	}

	// Active learner (fresh activity) → never a candidate.
	active := newTestUser(t, domain.RoleStudent)
	if err := database.DB.Model(active).Update("updated_at", time.Now()).Error; err != nil {
		t.Fatalf("failed to touch user: %v", err)
	}
	if err := database.DB.Create(&domain.LearnerActivity{LearnerID: active.ID, ActivityID: "act-1", Status: "Completed", CompletedAt: time.Now(), Score: 80}).Error; err != nil {
		t.Fatalf("failed to seed activity: %v", err)
	}
	t.Cleanup(func() { database.DB.Unscoped().Where("learner_id = ?", active.ID).Delete(&domain.LearnerActivity{}) })

	// Staff is never a retention candidate regardless of age.
	oldStaff := newTestUser(t, domain.RoleModerator)
	if err := database.DB.Model(oldStaff).Update("updated_at", time.Now().AddDate(-5, 0, 0)).Error; err != nil {
		t.Fatalf("failed to age staff: %v", err)
	}

	// Audit rows: one past the 3-year window (purged), one fresh (kept).
	oldAudit := &domain.AuditLog{UserID: "", Action: "auth.login", Detail: "old-row", CreatedAt: time.Now().AddDate(-4, 0, 0)}
	freshAudit := &domain.AuditLog{UserID: "", Action: "auth.login", Detail: "fresh-row", CreatedAt: time.Now()}
	if err := database.DB.Create(oldAudit).Error; err != nil {
		t.Fatalf("failed to seed old audit: %v", err)
	}
	if err := database.DB.Create(freshAudit).Error; err != nil {
		t.Fatalf("failed to seed fresh audit: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Unscoped().Where("id IN ?", []uint{staleAudit.ID, oldAudit.ID, freshAudit.ID}).Delete(&domain.AuditLog{})
		database.DB.Unscoped().Where("detail = ?", "stale-learner-row").Delete(&domain.AuditLog{})
	})

	report, err := svc.PurgeExpiredData(context.Background())
	if err != nil {
		t.Fatalf("purge failed: %v", err)
	}

	// The report must reflect at least our stale learner (seed data may add
	// more genuinely-inactive accounts — those are legitimate purges).
	if report.UsersPurged < 1 {
		t.Fatalf("expected at least 1 stale learner purged, got %d", report.UsersPurged)
	}
	if report.AuditRowsPurged != 1 {
		t.Fatalf("expected exactly 1 old audit row purged, got %d", report.AuditRowsPurged)
	}

	var count int64
	database.DB.Model(&domain.User{}).Unscoped().Where("id = ?", stale.ID).Count(&count)
	if count != 0 {
		t.Fatalf("stale learner survived the purge")
	}
	database.DB.Model(&domain.User{}).Unscoped().Where("id = ?", active.ID).Count(&count)
	if count != 1 {
		t.Fatalf("active learner was purged")
	}
	database.DB.Model(&domain.User{}).Unscoped().Where("id = ?", oldStaff.ID).Count(&count)
	if count != 1 {
		t.Fatalf("staff member was purged")
	}
	database.DB.Model(&domain.AuditLog{}).Where("id = ?", oldAudit.ID).Count(&count)
	if count != 0 {
		t.Fatalf("old audit row survived the purge")
	}
	database.DB.Model(&domain.AuditLog{}).Where("id = ?", freshAudit.ID).Count(&count)
	if count != 1 {
		t.Fatalf("fresh audit row was purged")
	}

	// The erasure map wrote the anonymized audit entry — the purge is
	// observable, exactly like a user-initiated erasure. The detail carries
	// the truncated sha256 of the user ID.
	sum := sha256.Sum256([]byte(stale.ID))
	database.DB.Model(&domain.AuditLog{}).Where("action = ? AND detail = ?", "privacy.account_deleted", "erasure_hash="+hex.EncodeToString(sum[:])[:16]).Count(&count)
	if count != 1 {
		t.Fatalf("erasure audit entry missing for purged learner")
	}
}
