package domain

import (
	"context"
	"time"
)

// UserRepository handles user persistence
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	FindByPhoneUnscoped(ctx context.Context, phone string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
}

// AuthRepository handles OTP and Token blocklists
type AuthRepository interface {
	SaveOTP(ctx context.Context, record *OTPRecord) error
	FindOTPByPhone(ctx context.Context, phone string) (*OTPRecord, error)
	IncrementOTPAttempts(ctx context.Context, phone string) error
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
	SyncBulk(ctx context.Context, learnerID string, data []SyncRequestItem) (int, int, error) // processed, failed
}

// CompletionRepository runs the activity-completion flow atomically so a
// partial failure never leaves inconsistent state (learner activity, progress,
// observation, and guidance are written or rolled back together).
type CompletionRepository interface {
	CompleteActivityTx(ctx context.Context, learnerID, activityID string, stats AttemptStats) (Observation, Guidance, error)
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
	GetRoster(ctx context.Context, page, limit int) ([]RosterEntry, int64, int64, error)
}

// SchoolRepository handles classes, enrollment, announcements, assignments,
// submissions, audit logging, and session revocation.
type SchoolRepository interface {
	CreateClass(ctx context.Context, class *Class) error
	FindClassByID(ctx context.Context, id string) (*Class, error)
	ListClasses(ctx context.Context) ([]Class, error)
	ListClassesByTeacher(ctx context.Context, teacherID string) ([]Class, error)
	Enroll(ctx context.Context, classID string, userIDs []string) error
	RemoveMember(ctx context.Context, classID, userID string) error
	ClassMembers(ctx context.Context, classID string) ([]User, error)
	ClassMemberCount(ctx context.Context, classID string) (int64, error)
	IsMember(ctx context.Context, classID, userID string) (bool, error)
	ClassesOfLearner(ctx context.Context, learnerID string) ([]Class, error)

	CreateAnnouncement(ctx context.Context, ann *Announcement) error
	ListAnnouncements(ctx context.Context, limit int) ([]Announcement, error)

	CreateAssignment(ctx context.Context, a *Assignment) error
	FindAssignmentByID(ctx context.Context, id string) (*Assignment, error)
	AssignmentsForClass(ctx context.Context, classID string) ([]Assignment, error)
	AssignmentsForLearner(ctx context.Context, learnerID string) ([]Assignment, error)

	SubmitAssignment(ctx context.Context, s *Submission) error // idempotent upsert on (assignment_id, learner_id)
	FindSubmission(ctx context.Context, assignmentID, learnerID string) (*Submission, error)
	SubmissionsForAssignment(ctx context.Context, assignmentID string) ([]Submission, error)
	SubmissionCount(ctx context.Context, assignmentID string) (int64, error)

	WriteAuditLog(ctx context.Context, entry *AuditLog) error
	ListAuditLogs(ctx context.Context, limit int) ([]AuditLog, error)

	RevokeAll(ctx context.Context, userID string, before time.Time) error
	RevokedBefore(ctx context.Context, userID string) (*UserRevocation, error)
}
