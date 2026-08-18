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
	CreateAnnouncement(ctx context.Context, title, body, authorID string) (*domain.Announcement, error)
	ListAnnouncements(ctx context.Context, limit int) ([]domain.Announcement, error)
	CreateAssignment(ctx context.Context, classID, title, description, activityID, createdBy string, dueDate string) (*domain.Assignment, error)
	AssignmentsForClass(ctx context.Context, classID string) ([]domain.Assignment, error)
	AssignmentsForLearner(ctx context.Context, learnerID string) ([]domain.Assignment, error)
	SubmitAssignment(ctx context.Context, assignmentID, learnerID, note string) (*domain.Submission, error)
	FindSubmission(ctx context.Context, assignmentID, learnerID string) (*domain.Submission, error)
	SubmissionsForAssignment(ctx context.Context, assignmentID string) ([]domain.Submission, error)
	SubmissionCount(ctx context.Context, assignmentID string) (int64, error)
	WriteAuditLog(ctx context.Context, userID, action, detail, ip string)
	ListAuditLogs(ctx context.Context, limit int) ([]domain.AuditLog, error)
	RevokeAll(ctx context.Context, userID string) error
	RevokedBefore(ctx context.Context, userID string) (*domain.UserRevocation, error)
}
