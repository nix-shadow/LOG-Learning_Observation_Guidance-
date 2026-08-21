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

func newParentTestHandler(t *testing.T) *ParentHandler {
	parentRepo := repository.NewParentRepository(database.DB)
	userRepo := repository.NewUserRepository(database.DB)
	schoolRepo := repository.NewSchoolRepository(database.DB)
	learnerService := service.NewLearnerService(
		userRepo,
		repository.NewActivityRepository(database.DB),
		repository.NewProgressRepository(database.DB),
		repository.NewLearnerDataRepository(database.DB),
		repository.NewCompletionRepository(database.DB),
	)
	parentService := service.NewParentService(parentRepo, userRepo, schoolRepo, learnerService)
	return NewParentHandler(parentService, service.NewSchoolService(schoolRepo))
}

// claimParentLinkForTest drives the full teacher-invite → guardian-claim
// flow over HTTP and returns the claimed PARENT user. Used to set up the
// digest/opt-in tests.
func claimParentLinkForTest(t *testing.T, teacherID, studentID string) *domain.User {
	h := newParentTestHandler(t)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", teacherID); c.Next() })
	r.POST("/api/v1/moderator/students/:id/parent-invite", h.CreateParentInvite)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/moderator/students/"+studentID+"/parent-invite", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create invite: %v: %s", w.Code, w.Body.String())
	}
	var invite struct {
		Code string `json:"invite_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &invite); err != nil {
		t.Fatalf("unmarshal invite: %v", err)
	}

	email := service.GenerateSecureID("p") + "@parent.local"
	disclosure := strings.Repeat("ab", 32) // 64-char sha256 hex shape
	r2 := gin.New()
	r2.POST("/api/v1/auth/parent-signup", h.ParentSignup)
	w2 := httptest.NewRecorder()
	body := `{"name":"Test Parent","email":"` + email + `","password":"Parent@123","invite_code":"` + invite.Code + `","disclosure_hash":"` + disclosure + `"}`
	req2, _ := http.NewRequest("POST", "/api/v1/auth/parent-signup", bytes.NewBufferString(body))
	r2.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("parent signup: %v: %s", w2.Code, w2.Body.String())
	}
	var res struct {
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal signup: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Unscoped().Where("id = ?", res.User.ID).Delete(&domain.User{})
		database.DB.Where("user_id = ?", res.User.ID).Delete(&domain.ConsentRecord{})
		database.DB.Where("parent_id = ?", res.User.ID).Delete(&domain.ParentLink{})
	})
	return &domain.User{ID: res.User.ID, Name: res.User.Name, Email: email, Role: domain.RoleParent}
}

// TestParentInviteAndClaim proves the school-verified invite flow: the
// teacher's invite is the verification, the claim records parent_access
// consent with the disclosure hash, and the code is strictly one-time
// (WP-2.1 wp21t01).
func TestParentInviteAndClaim(t *testing.T) {
	h := newParentTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)
	class := newClassWithTeacher(t, teacher.ID)
	if err := repository.NewSchoolRepository(database.DB).Enroll(context.Background(), class.ID, []string{student.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	enrollCleanup(t, class.ID, student.ID)

	// Teacher creates the invite — the school verification.
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", teacher.ID); c.Next() })
	r.POST("/api/v1/moderator/students/:id/parent-invite", h.CreateParentInvite)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/moderator/students/"+student.ID+"/parent-invite", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %v: %s", w.Code, w.Body.String())
	}
	var invite struct {
		Code string `json:"invite_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &invite); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(invite.Code) != 6 {
		t.Fatalf("expected 6-char invite code, got %q", invite.Code)
	}

	// A teacher outside the class gets a hard 404 — no invite leakage.
	other := newTestUser(t, domain.RoleModerator)
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("userID", other.ID); c.Next() })
	r2.POST("/api/v1/moderator/students/:id/parent-invite", h.CreateParentInvite)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/moderator/students/"+student.ID+"/parent-invite", nil)
	r2.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner teacher, got %v", w2.Code)
	}

	// Guardian claims the code: account + consent in one atomic flow.
	disclosure := strings.Repeat("cd", 32)
	r3 := gin.New()
	r3.POST("/api/v1/auth/parent-signup", h.ParentSignup)
	w3 := httptest.NewRecorder()
	body := `{"name":"Guardian Sita","email":"` + service.GenerateSecureID("p") + `@parent.local","password":"Parent@123","invite_code":"` + invite.Code + `","disclosure_hash":"` + disclosure + `","language":"ne"}`
	req3, _ := http.NewRequest("POST", "/api/v1/auth/parent-signup", bytes.NewBufferString(body))
	r3.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 claim, got %v: %s", w3.Code, w3.Body.String())
	}
	var res struct {
		Token string `json:"token"`
		User  struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w3.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal claim: %v", err)
	}
	if res.Token == "" || res.User.Role != string(domain.RoleParent) {
		t.Fatalf("expected PARENT token, got %+v", res)
	}

	// Consent evidence is real: parent_access + the exact notice hash.
	var consent domain.ConsentRecord
	if err := database.DB.First(&consent, "user_id = ? AND consent_type = ?", res.User.ID, domain.ConsentTypeParentAccess).Error; err != nil {
		t.Fatalf("parent_access consent missing: %v", err)
	}
	if consent.DisclosureHash != disclosure {
		t.Fatalf("disclosure hash drift: got %q want %q", consent.DisclosureHash, disclosure)
	}
	t.Cleanup(func() {
		database.DB.Unscoped().Where("id = ?", res.User.ID).Delete(&domain.User{})
		database.DB.Where("user_id = ?", res.User.ID).Delete(&domain.ConsentRecord{})
	})

	// The code is one-time: a second claim is an honest 404.
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("POST", "/api/v1/auth/parent-signup", bytes.NewBufferString(`{"name":"Second","email":"`+service.GenerateSecureID("p")+`@parent.local","password":"Parent@123","invite_code":"`+invite.Code+`","disclosure_hash":"`+disclosure+`"}`))
	r3.ServeHTTP(w4, req4)
	if w4.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for reused code, got %v: %s", w4.Code, w4.Body.String())
	}

	// Missing disclosure hash is rejected — unprovable grants are no grants.
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest("POST", "/api/v1/auth/parent-signup", bytes.NewBufferString(`{"name":"Third","email":"`+service.GenerateSecureID("p")+`@parent.local","password":"Parent@123","invite_code":"`+invite.Code+`"}`))
	r3.ServeHTTP(w5, req5)
	if w5.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without disclosure hash, got %v", w5.Code)
	}
}

// TestParentDigestReadOnlySanitized proves the read-only digest never leaks
// sensitive data: no email, no phone, no observations — only name, progress,
// activity status and guidance (WP-2.1 wp21t02).
func TestParentDigestReadOnlySanitized(t *testing.T) {
	h := newParentTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)
	class := newClassWithTeacher(t, teacher.ID)
	if err := repository.NewSchoolRepository(database.DB).Enroll(context.Background(), class.ID, []string{student.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	enrollCleanup(t, class.ID, student.ID)
	parent := claimParentLinkForTest(t, teacher.ID, student.ID)

	// Real data: an activity with stored accuracy/attempts + a private
	// observation that must never reach the parent view.
	act := &domain.Activity{ID: service.GenerateSecureID("act"), Title: "Nepali Essay", Topic: "Nepali"}
	if err := database.DB.Create(act).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	t.Cleanup(func() { database.DB.Where("id = ?", act.ID).Delete(&domain.Activity{}) })
	database.DB.Create(&domain.LearnerActivity{
		LearnerID:  student.ID,
		ActivityID: act.ID,
		Status:     domain.StatusNeedsPractice,
		Accuracy:   0.55,
		Attempts:   2,
	})
	t.Cleanup(func() { database.DB.Where("learner_id = ?", student.ID).Delete(&domain.LearnerActivity{}) })
	database.DB.Create(&domain.Observation{
		ID:        service.GenerateSecureID("obs"),
		LearnerID: student.ID,
		Text:      "PRIVATE-TEACHER-OBSERVATION",
	})
	t.Cleanup(func() { database.DB.Where("learner_id = ?", student.ID).Delete(&domain.Observation{}) })
	phone := "9800000099"
	database.DB.Model(&domain.User{}).Where("id = ?", student.ID).Update("phone", &phone)

	// Children list: minimal identity only.
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", parent.ID); c.Next() })
	r.GET("/api/v1/parents/children", h.ListChildren)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/parents/children", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("children: %v: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, student.ID) || !strings.Contains(body, student.Name) {
		t.Fatalf("expected linked child, got %s", body)
	}
	if strings.Contains(body, student.Email) || strings.Contains(body, phone) {
		t.Fatal("children list leaked email/phone")
	}

	// Digest: real progress + guidance, zero sensitive fields.
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("userID", parent.ID); c.Next() })
	r2.GET("/api/v1/parents/children/:id/digest", h.ChildDigest)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/parents/children/"+student.ID+"/digest", nil)
	r2.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("digest: %v: %s", w2.Code, w2.Body.String())
	}
	digestBody := w2.Body.String()
	if !strings.Contains(digestBody, act.Title) || !strings.Contains(digestBody, "needs-practice") {
		t.Fatalf("digest missing real progress: %s", digestBody)
	}
	if !strings.Contains(digestBody, "as_of") {
		t.Fatal("digest must carry an as_of freshness timestamp")
	}
	for _, leaked := range []string{student.Email, phone, "PRIVATE-TEACHER-OBSERVATION", "guardian@", "teacher@"} {
		if strings.Contains(digestBody, leaked) {
			t.Fatalf("digest leaked %q", leaked)
		}
	}

	// Unlinked parent → hard 404, never a glimpse. The stranger parent is
	// linked to a DIFFERENT learner (also in the teacher's class) so their
	// claim itself is valid — only the digest scope must fail.
	strangerStudent := newTestUser(t, domain.RoleStudent)
	if err := repository.NewSchoolRepository(database.DB).Enroll(context.Background(), class.ID, []string{strangerStudent.ID}); err != nil {
		t.Fatalf("enroll stranger: %v", err)
	}
	enrollCleanup(t, class.ID, strangerStudent.ID)
	stranger := claimParentLinkForTest(t, teacher.ID, strangerStudent.ID)
	r3 := gin.New()
	r3.Use(func(c *gin.Context) { c.Set("userID", stranger.ID); c.Next() })
	r3.GET("/api/v1/parents/children/:id/digest", h.ChildDigest)
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/v1/parents/children/"+student.ID+"/digest", nil)
	r3.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unlinked parent, got %v", w3.Code)
	}
}

// TestParentDigestOptIn proves the digest opt-in preference is the parent's
// own, reversible choice (WP-2.1 wp21t03).
func TestParentDigestOptIn(t *testing.T) {
	h := newParentTestHandler(t)
	teacher := newTestUser(t, domain.RoleModerator)
	student := newTestUser(t, domain.RoleStudent)
	class := newClassWithTeacher(t, teacher.ID)
	if err := repository.NewSchoolRepository(database.DB).Enroll(context.Background(), class.ID, []string{student.ID}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	enrollCleanup(t, class.ID, student.ID)
	parent := claimParentLinkForTest(t, teacher.ID, student.ID)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", parent.ID); c.Next() })
	r.POST("/api/v1/parents/children/:id/opt-in", h.SetDigestOptIn)
	r.GET("/api/v1/parents/children", h.ListChildren)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/parents/children/"+student.ID+"/opt-in", bytes.NewBufferString(`{"enabled":true}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("opt-in: %v: %s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/parents/children", nil)
	r.ServeHTTP(w2, req2)
	if !strings.Contains(w2.Body.String(), `"digest_opt_in":true`) {
		t.Fatalf("opt-in not reflected in children list: %s", w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/api/v1/parents/children/"+student.ID+"/opt-in", bytes.NewBufferString(`{"enabled":false}`))
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("opt-out: %v", w3.Code)
	}
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/api/v1/parents/children", nil)
	r.ServeHTTP(w4, req4)
	if !strings.Contains(w4.Body.String(), `"digest_opt_in":false`) {
		t.Fatalf("opt-out not reflected: %s", w4.Body.String())
	}
}
