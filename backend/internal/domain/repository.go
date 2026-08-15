package domain

import (
	"context"
)

// UserRepository handles user persistence
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
}

// AuthRepository handles OTP and Token blocklists
type AuthRepository interface {
	SaveOTP(ctx context.Context, record *OTPRecord) error
	FindOTPByPhone(ctx context.Context, phone string) (*OTPRecord, error)
	DeleteOTP(ctx context.Context, phone string) error
	DeleteExpiredOTPs(ctx context.Context) error
	
	BlockToken(ctx context.Context, blocklist *TokenBlocklist) error
	IsTokenBlocked(ctx context.Context, jti string) (bool, error)
}

// ActivityRepository handles activity logic
type ActivityRepository interface {
	FindAll(ctx context.Context) ([]Activity, error)
	FindByID(ctx context.Context, id string) (*Activity, error)
}

// ProgressRepository handles progress and learner activities
type ProgressRepository interface {
	FindLearnerActivities(ctx context.Context, learnerID string) ([]LearnerActivity, error)
	FindLearnerActivity(ctx context.Context, learnerID, activityID string) (*LearnerActivity, error)
	SaveLearnerActivity(ctx context.Context, la *LearnerActivity) error
	FindProgress(ctx context.Context, learnerID string) (*Progress, error)
	SaveProgress(ctx context.Context, progress *Progress) error
}

// SyncRepository handles bulk offline sync logic
type SyncRepository interface {
	SyncBulk(ctx context.Context, learnerID string, data []SyncRequestItem) (int, error)
}

// LearnerDataRepository handles other learner data like observations, guidance, and daily activities
type LearnerDataRepository interface {
	FindObservations(ctx context.Context, learnerID string) ([]Observation, error)
	FindGuidance(ctx context.Context, learnerID string) ([]Guidance, error)
	FindDailyActivities(ctx context.Context, learnerID string) ([]DailyActivity, error)
	SaveObservation(ctx context.Context, obs *Observation) error
	SaveGuidance(ctx context.Context, gui *Guidance) error
}

// CourseRepository handles courses and modules
type CourseRepository interface {
	FindCourses(ctx context.Context, page, limit int) ([]Course, int64, error)
	FindModulesByActivityID(ctx context.Context, activityID string) ([]MicroModule, error)
}

// ModeratorRepository handles moderator specific logic
type ModeratorRepository interface {
	GetRoster(ctx context.Context, page, limit int) ([]User, int64, int64, error)
}
