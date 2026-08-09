package models

import "time"

type Learner struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Activity struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` // "Not started", "In progress", "Completed"
	Topic       string `json:"topic"`
	Order       int    `json:"order"`
}

type Progress struct {
	LearnerID     string  `json:"learner_id"`
	TotalTopics   int     `json:"total_topics"`
	Completed     int     `json:"completed"`
	CurrentStreak int     `json:"current_streak"`
	OverallScore  float64 `json:"overall_score"`
}

type Observation struct {
	ID        string    `json:"id"`
	LearnerID string    `json:"learner_id"`
	Category  string    `json:"category"` // "progress", "consistency", "strengths", "areas needing improvement"
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type Guidance struct {
	ID        string    `json:"id"`
	LearnerID string    `json:"learner_id"`
	Text      string    `json:"text"`
	Action    string    `json:"action"`
	Type      string    `json:"type"` // "next_step", "practice", "insight"
	CreatedAt time.Time `json:"created_at"`
}
