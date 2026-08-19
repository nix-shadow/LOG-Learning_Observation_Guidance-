package domain

import (
	"context"
	"time"
)

// UserRepository handles user persistence
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
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
}

// ProgressRepository handles progress and learner activities
type ProgressRepository interface {
	FindLearnerActivities(ctx context.Context, learnerID string) ([]LearnerActivity, error)
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
}

// CourseRepository handles courses, modules, and per-learner enrollment.
type CourseRepository interface {
	// FindCourses returns one page, annotated with the real enrollment count
	// and the caller's own is_enrolled flag (WP-0.2 C5 — no invented numbers).
	FindCourses(ctx context.Context, userID string, page, limit int) ([]Course, int64, error)
	FindModulesByActivityID(ctx context.Context, activityID string) ([]MicroModule, error)
	Enroll(ctx context.Context, userID, courseID string) error
	Unenroll(ctx context.Context, userID, courseID string) error
}

// ModeratorRepository handles moderator specific logic. The roster is scoped
// to the caller's own classes: a teacher never sees another teacher's students.
type ModeratorRepository interface {
	GetRoster(ctx context.Context, teacherID string, page, limit int) ([]RosterEntry, int64, int64, error)
	AssignmentsDueForTeacher(ctx context.Context, teacherID string) (int64, error)
	FirstClassNameForTeacher(ctx context.Context, teacherID string) (string, error)
}

// SchoolRepository handles classes, enrollment, announcements, assignments,
// submissions, audit logging, and session revocation.
type SchoolRepository interface {
	CreateClass(ctx context.Context, class *Class) error
	FindClassByID(ctx context.Context, id string) (*Class, error)
	ListClasses(ctx context.Context) ([]Class, error)
	ListClassesByTeacher(ctx context.Context, teacherID string) ([]Class, error)
	Enroll(ctx context.Context, classID string, userIDs []string) error
	RemoveMember(ctx context.Context, classID, userID string) (int64, error)
	ClassMembers(ctx context.Context, classID string) ([]User, error)
	ClassMemberCount(ctx context.Context, classID string) (int64, error)
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
	SubmissionCounts(ctx context.Context, assignmentIDs []string) (map[string]int64, error)

	WriteAuditLog(ctx context.Context, entry *AuditLog) error
	// ListAuditLogs returns one page (limit, offset) — never an unbounded row
	// load (WP-0.2 C1). The total is returned so the admin UI can page.
	ListAuditLogs(ctx context.Context, limit, offset int) ([]AuditLog, int64, error)

	RevokeAll(ctx context.Context, userID string, before time.Time) error
	RevokedBefore(ctx context.Context, userID string) (*UserRevocation, error)
}

// PrivacyRepository handles consent records, personal-data export, and
// account erasure (the deletion data map).
type PrivacyRepository interface {
	// UpsertConsent records or refreshes a consent row (one active row per
	// user + type). Re-granting bumps Version/GrantedAt; withdrawal flips status.
	UpsertConsent(ctx context.Context, record *ConsentRecord) error
	GetConsents(ctx context.Context, userID string) ([]ConsentRecord, error)
	// GetConsentMap returns the latest granted consent of the given type for
	// each of the user IDs (used by the admin users list).
	GetConsentMap(ctx context.Context, userIDs []string, consentType string) (map[string]ConsentRecord, error)
	// HasActiveConsent reports whether the user has a currently-granted (not
	// withdrawn) consent of the given type. Used by the server-side consent
	// gate so a learner cannot mutate data without guardian consent, even if
	// the login UI is bypassed.
	HasActiveConsent(ctx context.Context, userID, consentType string) (bool, error)
	// FindInactiveStudentIDs returns learner-role user IDs whose last activity
	// predates the cutoff — candidates for the retention purge.
	FindInactiveStudentIDs(ctx context.Context, olderThan time.Time) ([]string, error)
	// DeleteOldAuditLogs removes audit rows older than the retention window.
	DeleteOldAuditLogs(ctx context.Context, olderThan time.Time) (int64, error)
	// ExportData gathers every table that holds the user's own data.
	ExportData(ctx context.Context, userID string) (*ExportBundle, error)
	// DeleteAccountTx erases the user's data per the deletion data map and
	// hard-deletes the account. The audit entry (anonymized, UserID="") is
	// written inside the same transaction so the erasure trail is atomic with
	// the erasure.
	DeleteAccountTx(ctx context.Context, userID string, audit *AuditLog) error
	// ScrubDeletedData physically wipes logically-deleted rows: WAL checkpoint
	// (TRUNCATE) followed by VACUUM. Best-effort — callers log on failure. The
	// returned report carries the before/after evidence (freelist pages, WAL
	// frames) so the caller can log that the wipe actually shrank the surface.
	ScrubDeletedData(ctx context.Context) (*ScrubReport, error)
}
