package models

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
	ID          string         `json:"id" gorm:"primaryKey"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `json:"status"`
	Topic       string         `json:"topic"`
	Order       int            `json:"order"`
	ContentJSON string         `json:"content_json"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
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
	LearnerID string    `json:"learner_id" gorm:"index"`
	Category  string    `json:"category"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type Guidance struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	LearnerID string    `json:"learner_id" gorm:"index"`
	Text      string    `json:"text"`
	Action    string    `json:"action"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type SystemAnalytics struct {
	TotalUsers       int `json:"total_users"`
	ActiveDaily      int `json:"active_daily"`
	TotalCompletions int `json:"total_completions"`
}
