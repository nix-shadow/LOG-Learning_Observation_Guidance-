package domain

import (
	"time"

	"gorm.io/gorm"
)

type Role string

const (
	RoleStudent   Role = "STUDENT"
	RoleModerator Role = "MODERATOR" // Teacher
	RoleAdmin     Role = "ADMIN"     // Principal/HOD
)

type User struct {
	ID           string         `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name"`
	Email        string         `json:"email" gorm:"uniqueIndex"`
	Phone        *string        `json:"phone" gorm:"uniqueIndex"`
	PasswordHash string         `json:"-"`
	Role         Role           `json:"role"`
	IsVerified   bool           `json:"is_verified"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

type OTPRecord struct {
	Phone     string    `json:"phone" gorm:"primaryKey"`
	OTP       string    `json:"otp"`
	ExpiresAt time.Time `json:"expires_at"`
	Attempts  int       `json:"attempts"` // failed verify attempts — 5 fails invalidates the OTP
}

type Activity struct {
	ID            string `json:"id" gorm:"primaryKey"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Topic         string `json:"topic"`
	Order         int    `json:"order"`
	ContentJSON   string `json:"content_json"`
	Difficulty    string `json:"difficulty"`    // e.g. "Beginner", "Intermediate", "Advanced"
	Prerequisites string `json:"prerequisites"` // comma-separated list of required Activity IDs
	// WP-3.1 (RC-07): OER metadata — every activity must carry an honest
	// license + attribution so remixed packs never masquerade as own work.
	// "Own work (LOG team)" marks original content; everything else must be a
	// license from OERAllowedLicenses (validated at import time).
	License     string `json:"license"`
	LicenseURL  string `json:"license_url"`
	Attribution string `json:"attribution"`
	SourceURL   string `json:"source_url"`
	// WP-3.4 (RC-12): NSL caption text (plaintext description for the
	// sign-language caption track). Empty means none exists — the UI shows
	// nothing rather than inventing a caption.
	CaptionText string         `json:"caption_text"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// OER licenses the import pipeline accepts (WP-3.1). Anything else is
// rejected with a row-level error — the pipeline never guesses a license.
var OERAllowedLicenses = map[string]struct{}{
	"CC BY 4.0":               {},
	"CC BY-SA 4.0":            {},
	"CC BY-NC 4.0":            {},
	"CC BY-NC-SA 4.0":         {},
	"CC0 1.0 (Public Domain)": {},
	"Own work (LOG team)":     {},
}

// OERLicenseURLs maps each allowed license to its canonical deed URL so the
// UI can render a real link instead of a label alone.
var OERLicenseURLs = map[string]string{
	"CC BY 4.0":               "https://creativecommons.org/licenses/by/4.0/",
	"CC BY-SA 4.0":            "https://creativecommons.org/licenses/by-sa/4.0/",
	"CC BY-NC 4.0":            "https://creativecommons.org/licenses/by-nc/4.0/",
	"CC BY-NC-SA 4.0":         "https://creativecommons.org/licenses/by-nc-sa/4.0/",
	"CC0 1.0 (Public Domain)": "https://creativecommons.org/publicdomain/zero/1.0/",
	"Own work (LOG team)":     "",
}

// OERPack is one batch import payload (WP-3.1). Each activity must carry its
// own id + a license from OERAllowedLicenses; third-party content must also
// name its author in Attribution.
type OERPack struct {
	Name       string     `json:"name"`
	Activities []Activity `json:"activities"`
}

// OERImportReport is the honest outcome of a pack import: how many rows were
// imported, skipped (existing id), and rejected (with per-row reasons).
type OERImportReport struct {
	Imported int                 `json:"imported"`
	Skipped  int                 `json:"skipped"`
	Errors   []OERImportRowError `json:"errors"`
}

type OERImportRowError struct {
	Row    int    `json:"row"`
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

func IsAllowedOERLicense(license string) bool {
	_, ok := OERAllowedLicenses[license]
	return ok
}

type LearnerActivity struct {
	LearnerID  string `json:"learner_id" gorm:"primaryKey"`
	ActivityID string `json:"activity_id" gorm:"primaryKey"`
	Status     string `json:"status"` // canonical status — see Status* constants
	// NOTE (WP-4.2 study): no index on CompletedAt — the dashboard's
	// active-daily query (DISTINCT learner_id WHERE completed_at > ?) is
	// served by the PK index scan at every realistic school scale; a
	// dedicated index changed neither the plan nor runtime up to 200k rows
	// (see docs/QUERY_PLANS.md §Dashboard).
	CompletedAt    time.Time `json:"completed_at"`
	Score          float64   `json:"score"` // best attempt score (0-100), derived from real quiz accuracy
	Accuracy       float64   `json:"accuracy"`
	ElapsedSeconds int       `json:"elapsed_seconds"`
	Attempts       int       `json:"attempts"`
}

// Canonical per-learner activity statuses (WP-1.1 RC-01). Supportive by
// design: the vocabulary has no "failed" — a low-accuracy completion is
// "needs-practice", a nudge to improve, never a verdict.
const (
	StatusNotStarted    = "not-started"
	StatusActive        = "active"
	StatusNeedsPractice = "needs-practice"
	StatusCompleted     = "completed"

	// NeedsPracticeAccuracyThreshold: completing below this quiz accuracy
	// flags the activity for another supportive practice round.
	NeedsPracticeAccuracyThreshold = 0.7
)

// ResolveActivityStatus maps a stored LearnerActivity row to the canonical
// per-learner status vocabulary. It normalizes legacy rows ("Pending",
// "In Progress", "Completed") and derived states (attempts without a
// completion are "active"). A legacy "Completed" row with quiz accuracy
// below the threshold is honestly re-flagged "needs-practice".
func ResolveActivityStatus(la LearnerActivity) string {
	switch la.Status {
	case StatusNeedsPractice, StatusCompleted:
		return la.Status
	case "In Progress", "In progress", "Active", "active":
		return StatusActive
	case "Completed":
		if la.Accuracy > 0 && la.Accuracy < NeedsPracticeAccuracyThreshold {
			return StatusNeedsPractice
		}
		return StatusCompleted
	default:
		if la.Attempts > 0 {
			return StatusActive
		}
		return StatusNotStarted
	}
}

type MicroModule struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	ActivityID   string    `json:"activity_id" gorm:"index"`
	Title        string    `json:"title"`
	ContentText  string    `json:"content_text"` // extremely compressed text
	MediaURL     string    `json:"media_url"`    // optional low-res WebP image
	Question     string    `json:"question"`     // optional knowledge check
	Options      []string  `json:"options" gorm:"serializer:json"`
	CorrectIndex int       `json:"correct_index"`
	Explanation  string    `json:"explanation"` // supportive feedback shown after answering
	Order        int       `json:"order"`
	CreatedAt    time.Time `json:"created_at"`
}

type Progress struct {
	LearnerID        string    `json:"learner_id" gorm:"primaryKey"`
	TotalTopics      int       `json:"total_topics"`
	Completed        int       `json:"completed"`
	CurrentStreak    int       `json:"current_streak"`
	OverallScore     float64   `json:"overall_score"`
	LastActivityDate time.Time `json:"last_activity_date"` // streak math is date-aware
}

// RosterEntry pairs a roster student with their progress in a single
// repository call — the repository batches the progress lookup so the
// moderator roster never issues per-learner queries.
type RosterEntry struct {
	User     User     `json:"user"`
	Progress Progress `json:"progress"`
}

type Observation struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	LearnerID string    `json:"learner_id" gorm:"index:idx_learner_created"`
	Category  string    `json:"category"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at" gorm:"index:idx_learner_created"`
}

type Guidance struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	LearnerID string    `json:"learner_id" gorm:"index:idx_guidance_learner"`
	Text      string    `json:"text"`
	Action    string    `json:"action"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at" gorm:"index:idx_guidance_learner"`
}

type SystemAnalytics struct {
	TotalUsers       int `json:"total_users"`
	ActiveDaily      int `json:"active_daily"`
	TotalCompletions int `json:"total_completions"`
}

// TokenBlocklist stores revoked JWT IDs so that logged-out tokens are rejected
// even before their natural expiry time.
type TokenBlocklist struct {
	JTI       string    `json:"jti" gorm:"primaryKey"`   // JWT ID claim
	UserID    string    `json:"user_id" gorm:"index"`    // which user revoked
	ExpiresAt time.Time `json:"expires_at" gorm:"index"` // mirrors JWT exp — for cleanup
	RevokedAt time.Time `json:"revoked_at"`
}

type Course struct {
	ID         string         `json:"id" gorm:"primaryKey"`
	Title      string         `json:"title"`
	Category   string         `json:"category"`
	Difficulty string         `json:"difficulty"`
	Duration   string         `json:"duration"`
	Rating     float64        `json:"rating"`
	Enrolled   int            `json:"enrolled"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
	// IsEnrolled is per-learner state, derived from the enrollments table at
	// query time — never stored on the course row (WP-0.2 C5).
	IsEnrolled bool `json:"is_enrolled" gorm:"-"`
}

// Enrollment is the persisted, per-learner enrollment state (WP-0.2 C5).
// One row per (user, course); catalog counts and is_enrolled derive from
// these rows — the dashboard goal ring and course cards must never show
// invented numbers, and a static int on Course was exactly that.
type Enrollment struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"uniqueIndex:uq_enrollment;index"`
	CourseID  string    `json:"course_id" gorm:"uniqueIndex:uq_enrollment;index"`
	CreatedAt time.Time `json:"created_at"`
}

type DailyActivity struct {
	ID string `json:"id" gorm:"primaryKey"`
	// WP-4.2: composite (learner_id, date) index serves the chart-data query
	// (WHERE learner_id = ? ORDER BY date asc) in one scan.
	LearnerID string    `json:"learner_id" gorm:"index:idx_da_learner_date,priority:1"`
	Date      time.Time `json:"date" gorm:"index:idx_da_learner_date,priority:2"`
	DayName   string    `json:"name"` // e.g. "Mon"
	Score     float64   `json:"score"`
	Duration  int       `json:"duration" gorm:"not null;default:0"` // Ensure no NULLs are written
	// WP-1.2 RC-02 practice metrics: real counts and mean accuracy, never
	// fabricated. Both only advance from actual completions.
	Attempts int     `json:"attempts" gorm:"not null;default:0"`
	Accuracy float64 `json:"accuracy"` // 0-100 weighted mean of daily completions
}

// SyncRequestItem represents an offline sync request
type SyncRequestItem struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
	Body     string `json:"body"`
}

// Class is a school class/section (e.g. "Grade 10 A") owned by a teacher (MODERATOR).
type Class struct {
	ID         string         `json:"id" gorm:"primaryKey"`
	Name       string         `json:"name" gorm:"not null"`
	Grade      string         `json:"grade"`
	Section    string         `json:"section"`
	TeacherID  string         `json:"teacher_id" gorm:"index"`    // MODERATOR who owns the class
	InviteCode string         `json:"invite_code" gorm:"size:10"` // WP-1.5: join-by-code; uniqueness enforced in service (legacy rows may be empty)
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// ClassMember is an enrollment record: a STUDENT belonging to a Class.
type ClassMember struct {
	ClassID  string    `json:"class_id" gorm:"primaryKey"`
	UserID   string    `json:"user_id" gorm:"primaryKey"`
	JoinedAt time.Time `json:"joined_at"`
}

// Announcement is a school-wide notice visible to every authenticated role.
type Announcement struct {
	ID       string `json:"id" gorm:"primaryKey"`
	Title    string `json:"title" gorm:"not null"`
	Body     string `json:"body"`
	AuthorID string `json:"author_id" gorm:"index"`
	// WP-4.2: created_at index serves the newest-first listing without a
	// temp b-tree sort (see docs/QUERY_PLANS.md §Announcements).
	CreatedAt time.Time `json:"created_at" gorm:"index:idx_ann_created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Assignment is a task a teacher sets for a whole class. It may link to an
// Activity so learners can study toward it before submitting.
type Assignment struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	ClassID     string    `json:"class_id" gorm:"index"`
	Title       string    `json:"title" gorm:"not null"`
	Description string    `json:"description"`
	ActivityID  string    `json:"activity_id"` // optional linked activity
	DueDate     time.Time `json:"due_date"`
	CreatedBy   string    `json:"created_by" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Submission is a learner's answer to an Assignment. (AssignmentID, LearnerID)
// is unique so offline replays are idempotent — resubmission updates in place.
type Submission struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	AssignmentID string    `json:"assignment_id" gorm:"uniqueIndex:idx_sub_assignment_learner"`
	LearnerID    string    `json:"learner_id" gorm:"uniqueIndex:idx_sub_assignment_learner"`
	Note         string    `json:"note"`
	SubmittedAt  time.Time `json:"submitted_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AuditLog is an append-only trail of sensitive operations (role changes,
// class/announcement/assignment creation, exports, logout-all).
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    string    `json:"user_id" gorm:"index"`
	Action    string    `json:"action" gorm:"index"`
	Detail    string    `json:"detail"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// UserRevocation supports "log out on all devices": tokens issued before
// RevokedBefore are rejected even though they have not expired.
type UserRevocation struct {
	UserID        string    `json:"user_id" gorm:"primaryKey"`
	RevokedBefore time.Time `json:"revoked_before"`
}

// PilotScan records one QR poster scan (WP-3.3 RC-10). No IP, device id, or
// other personal data is persisted — a scan is just a poster id + moment.
// Started flips when the scanned learner clicks through to the activity, so
// the pilot measures an honest first-session drop-off (scans vs starts).
type PilotScan struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PosterID  string    `json:"poster_id" gorm:"index"` // Activity ID the poster links to
	Source    string    `json:"source"`                 // "qr" | "poster"
	Started   bool      `json:"started"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

// PilotStats carries only real, stored numbers — derived from PilotScan rows,
// never fabricated (AGENTS.md §1). StartRate is 0 when nothing started.
type PilotStats struct {
	TotalScans      int64              `json:"total_scans"`
	ScansToday      int64              `json:"scans_today"`
	Starts          int64              `json:"starts"`
	DistinctPosters int64              `json:"distinct_posters"`
	StartRate       float64            `json:"start_rate"`
	PerPoster       []PilotPosterStats `json:"per_poster"`
}

type PilotPosterStats struct {
	PosterID string `json:"poster_id"`
	Scans    int64  `json:"scans"`
	Starts   int64  `json:"starts"`
}
