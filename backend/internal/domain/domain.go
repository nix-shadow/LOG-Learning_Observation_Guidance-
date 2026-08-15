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
	Phone        string         `json:"phone" gorm:"uniqueIndex"`
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
	LearnerID   string    `json:"learner_id" gorm:"primaryKey"`
	ActivityID  string    `json:"activity_id" gorm:"primaryKey"`
	Status      string    `json:"status"` // e.g. "Completed", "Pending", "In Progress"
	CompletedAt time.Time `json:"completed_at"`
	Score       float64   `json:"score"`
}

type MicroModule struct {
	ID          string         `json:"id" gorm:"primaryKey"`
	ActivityID  string         `json:"activity_id" gorm:"index"`
	Title       string         `json:"title"`
	ContentText string         `json:"content_text"` // extremely compressed text
	MediaURL    string         `json:"media_url"`    // optional low-res WebP image
	Order       int            `json:"order"`
	CreatedAt   time.Time      `json:"created_at"`
}

type Progress struct {
	LearnerID     string  `json:"learner_id" gorm:"primaryKey"`
	TotalTopics   int     `json:"total_topics"`
	Completed     int     `json:"completed"`
	CurrentStreak int     `json:"current_streak"`
	OverallScore  float64 `json:"overall_score"`
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
	JTI       string    `json:"jti" gorm:"primaryKey"`           // JWT ID claim
	UserID    string    `json:"user_id" gorm:"index"`            // which user revoked
	ExpiresAt time.Time `json:"expires_at" gorm:"index"`         // mirrors JWT exp — for cleanup
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
