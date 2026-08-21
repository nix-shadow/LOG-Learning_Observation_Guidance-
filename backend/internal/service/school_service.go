package service

import (
	"context"

	"log-backend/internal/domain"
)

type SchoolService interface {
	CreateClass(ctx context.Context, name, grade, section, teacherID string) (*domain.Class, error)
	ListClasses(ctx context.Context) ([]domain.Class, error)
	ClassesByTeacher(ctx context.Context, teacherID string) ([]domain.Class, error)
	EnrollStudents(ctx context.Context, classID string, userIDs []string) (int, error)
	UnenrollStudent(ctx context.Context, classID, userID string) error
	ClassRoster(ctx context.Context, classID string) ([]domain.User, error)
	ClassMemberCount(ctx context.Context, classID string) (int64, error)
	// JoinClassByCode (WP-1.5): enrolls the learner in the class behind the
	// invite code; idempotent for existing members.
	JoinClassByCode(ctx context.Context, code, learnerID string) (*domain.Class, error)
	// ImportRoster (WP-1.5): creates/enrolls STUDENT users from CSV rows.
	// Honest per-row report — never silently drops a row.
	ImportRoster(ctx context.Context, classID, teacherID string, rows []RosterImportRow) (RosterImportReport, error)
	// StudentInTeacherClasses (WP-1.5): scope check for per-student progress.
	StudentInTeacherClasses(ctx context.Context, teacherID, learnerID string) (bool, error)
	CreateAnnouncement(ctx context.Context, title, body, authorID string) (*domain.Announcement, error)
	ListAnnouncements(ctx context.Context, limit int) ([]domain.Announcement, error)
	CreateAssignment(ctx context.Context, classID, title, description, activityID, createdBy string, dueDate string) (*domain.Assignment, error)
	AssignmentsForClass(ctx context.Context, classID, callerID string, isAdmin bool) ([]domain.Assignment, error)
	AssignmentsForLearner(ctx context.Context, learnerID string) ([]domain.Assignment, error)
	SubmitAssignment(ctx context.Context, assignmentID, learnerID, note string) (*domain.Submission, error)
	FindSubmission(ctx context.Context, assignmentID, learnerID string) (*domain.Submission, error)
	SubmissionsForAssignment(ctx context.Context, assignmentID, callerID string, isAdmin bool) ([]domain.Submission, error)
	SubmissionCount(ctx context.Context, assignmentID string) (int64, error)
	SubmissionCounts(ctx context.Context, assignmentIDs []string) (map[string]int64, error)
	WriteAuditLog(ctx context.Context, userID, action, detail, ip string)
	ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AuditLog, int64, error)
	RevokeAll(ctx context.Context, userID string) error
	RevokedBefore(ctx context.Context, userID string) (*domain.UserRevocation, error)
}

// RosterImportRow is one parsed CSV line for the WP-1.5 roster import.
// Password is OPTIONAL: when empty, the service generates a temporary one
// returned exactly once in RosterImportReport.Passwords (never logged).
type RosterImportRow struct {
	RowNo    int // 1-based line in the original file (header = 1)
	Name     string
	Email    string
	Phone    string
	Password string
}

// RosterImportReport is the honest outcome of an import: every skipped or
// failed row carries a human reason, and generated passwords are returned
// once (email → password) so the teacher can hand them out.
type RosterImportReport struct {
	Imported  int               `json:"imported"`
	Skipped   int               `json:"skipped"`
	Passwords map[string]string `json:"passwords"` // email → generated temp password (one-time)
	Errors    []RosterRowError  `json:"errors"`    // per-row failures with reasons
}

type RosterRowError struct {
	Row    int    `json:"row"`
	Email  string `json:"email"`
	Reason string `json:"reason"`
}
