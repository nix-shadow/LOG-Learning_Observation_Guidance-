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
	ID            string         `json:"id" gorm:"primaryKey"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Topic         string         `json:"topic"`
	Order         int            `json:"order"`
	ContentJSON   string         `json:"content_json"`
	Difficulty    string         `json:"difficulty"`    // e.g. "Beginner", "Intermediate", "Advanced"
	Prerequisites string         `json:"prerequisites"` // comma-separated list of required Activity IDs
	CreatedAt     time.Time      `json:"created_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type LearnerActivity struct {
	LearnerID      string    `json:"learner_id" gorm:"primaryKey"`
	ActivityID     string    `json:"activity_id" gorm:"primaryKey"`
	Status         string    `json:"status"` // e.g. "Completed", "Pending", "In Progress"
	CompletedAt    time.Time `json:"completed_at"`
	Score          float64   `json:"score"` // best attempt score (0-100), derived from real quiz accuracy
	Accuracy       float64   `json:"accuracy"`
	ElapsedSeconds int       `json:"elapsed_seconds"`
	Attempts       int       `json:"attempts"`
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
	ID        string    `json:"id" gorm:"primaryKey"`
	LearnerID string    `json:"learner_id" gorm:"index"`
	Date      time.Time `json:"date" gorm:"index"`
	DayName   string    `json:"name"` // e.g. "Mon"
	Score     float64   `json:"score"`
	Duration  int       `json:"duration" gorm:"not null;default:0"` // Ensure no NULLs are written
}

// SyncRequestItem represents an offline sync request
type SyncRequestItem struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
	Body     string `json:"body"`
}

// Class is a school class/section (e.g. "Grade 10 A") owned by a teacher (MODERATOR).
type Class struct {
	ID        string         `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"not null"`
	Grade     string         `json:"grade"`
	Section   string         `json:"section"`
	TeacherID string         `json:"teacher_id" gorm:"index"` // MODERATOR who owns the class
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// ClassMember is an enrollment record: a STUDENT belonging to a Class.
type ClassMember struct {
	ClassID  string    `json:"class_id" gorm:"primaryKey"`
	UserID   string    `json:"user_id" gorm:"primaryKey"`
	JoinedAt time.Time `json:"joined_at"`
}

// Announcement is a school-wide notice visible to every authenticated role.
type Announcement struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title" gorm:"not null"`
	Body      string    `json:"body"`
	AuthorID  string    `json:"author_id" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
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
