package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"log-backend/internal/domain"
)

// Sentinel errors for school operations
var (
	ErrNotClassTeacher    = errors.New("only the class teacher may perform this action")
	ErrNotClassMember     = errors.New("you are not enrolled in this class")
	ErrNotEnrolled        = errors.New("student is not enrolled in this class")
	ErrInvalidDueDate     = errors.New("invalid due date, use RFC 3339 format")
	ErrClassNotFound      = errors.New("class not found")
	ErrAssignmentNotFound = errors.New("assignment not found")
	ErrInviteCodeNotFound = errors.New("no class found for this invite code")
)

type schoolService struct {
	repo domain.SchoolRepository
}

func NewSchoolService(repo domain.SchoolRepository) SchoolService {
	return &schoolService{repo: repo}
}

func (s *schoolService) CreateClass(ctx context.Context, name, grade, section, teacherID string) (*domain.Class, error) {
	// Generate a collision-free invite code (uniqueness is enforced here, not
	// by a DB index, so legacy rows without codes never break migrations).
	code := GenerateInviteCode()
	for i := 0; i < 5; i++ {
		if _, err := s.repo.FindClassByInviteCode(ctx, code); err != nil {
			break
		}
		code = GenerateInviteCode()
	}
	class := &domain.Class{
		ID:         GenerateSecureID("cls"),
		Name:       name,
		Grade:      grade,
		Section:    section,
		TeacherID:  teacherID,
		InviteCode: code,
		CreatedAt:  time.Now(),
	}
	if err := s.repo.CreateClass(ctx, class); err != nil {
		return nil, err
	}
	return class, nil
}

// JoinClassByCode (WP-1.5): enrolls the learner in the class behind the code.
// Idempotent — an existing member gets the class back, not an error.
func (s *schoolService) JoinClassByCode(ctx context.Context, code, learnerID string) (*domain.Class, error) {
	class, err := s.repo.FindClassByInviteCode(ctx, code)
	if err != nil {
		return nil, ErrInviteCodeNotFound
	}
	if err := s.repo.Enroll(ctx, class.ID, []string{learnerID}); err != nil {
		return nil, err
	}
	return class, nil
}

// StudentInTeacherClasses (WP-1.5): scope gate for per-student progress.
func (s *schoolService) StudentInTeacherClasses(ctx context.Context, teacherID, learnerID string) (bool, error) {
	return s.repo.StudentInTeacherClasses(ctx, teacherID, learnerID)
}

// ImportRoster (WP-1.5): find-or-create STUDENT users from CSV rows and
// enroll them. Every skipped/failed row is reported honestly; generated
// passwords are returned exactly once.
func (s *schoolService) ImportRoster(ctx context.Context, classID, teacherID string, rows []RosterImportRow) (RosterImportReport, error) {
	report := RosterImportReport{Passwords: map[string]string{}}
	class, err := s.repo.FindClassByID(ctx, classID)
	if err != nil {
		return report, ErrClassNotFound
	}
	if class.TeacherID != teacherID {
		return report, ErrNotClassTeacher
	}

	var toEnroll []string
	for _, row := range rows {
		line := row.RowNo
		if line == 0 {
			line = 1
		}
		email := strings.TrimSpace(row.Email)
		name := strings.TrimSpace(row.Name)
		if email == "" || !strings.Contains(email, "@") {
			report.Errors = append(report.Errors, RosterRowError{Row: line, Email: email, Reason: "missing or invalid email"})
			continue
		}
		if name == "" {
			report.Errors = append(report.Errors, RosterRowError{Row: line, Email: email, Reason: "missing name"})
			continue
		}

		user, err := s.repo.FindUserByEmail(ctx, email)
		if err != nil {
			// New student: create with a hashed password (given or generated).
			password := strings.TrimSpace(row.Password)
			if password == "" {
				password = GenerateTempPassword()
			}
			hash, err := HashPassword(password)
			if err != nil {
				report.Errors = append(report.Errors, RosterRowError{Row: line, Email: email, Reason: "could not hash password"})
				continue
			}
			phone := strings.TrimSpace(row.Phone)
			user = &domain.User{
				ID:           GenerateSecureID("usr"),
				Name:         name,
				Email:        email,
				Phone:        &phone,
				PasswordHash: hash,
				Role:         domain.RoleStudent,
			}
			if err := s.repo.CreateStudentUser(ctx, user); err != nil {
				// Honest per-row failure (e.g. duplicate phone) — never a
				// silent drop and never a fabricated success.
				report.Errors = append(report.Errors, RosterRowError{Row: line, Email: email, Reason: "could not create user"})
				continue
			}
			report.Passwords[email] = password
			report.Imported++
			toEnroll = append(toEnroll, user.ID)
			continue
		}

		// Existing user: enroll if they are a student; otherwise skip honestly.
		if user.Role != domain.RoleStudent {
			report.Skipped++
			report.Errors = append(report.Errors, RosterRowError{Row: line, Email: email, Reason: "existing user is not a student — skipped"})
			continue
		}
		toEnroll = append(toEnroll, user.ID)
		report.Imported++
	}

	if err := s.repo.Enroll(ctx, class.ID, toEnroll); err != nil {
		return report, err
	}
	return report, nil
}

func (s *schoolService) ListClasses(ctx context.Context) ([]domain.Class, error) {
	return s.repo.ListClasses(ctx)
}

func (s *schoolService) ClassesByTeacher(ctx context.Context, teacherID string) ([]domain.Class, error) {
	return s.repo.ListClassesByTeacher(ctx, teacherID)
}

func (s *schoolService) EnrollStudents(ctx context.Context, classID string, userIDs []string) (int, error) {
	if _, err := s.repo.FindClassByID(ctx, classID); err != nil {
		return 0, ErrClassNotFound
	}
	if len(userIDs) == 0 {
		return 0, nil
	}
	if err := s.repo.Enroll(ctx, classID, userIDs); err != nil {
		return 0, err
	}
	count, err := s.repo.ClassMemberCount(ctx, classID)
	return int(count), err
}

func (s *schoolService) UnenrollStudent(ctx context.Context, classID, userID string) error {
	removed, err := s.repo.RemoveMember(ctx, classID, userID)
	if err != nil {
		return err
	}
	// A successful no-op is still a lie: report honestly when the student was
	// never enrolled in the first place.
	if removed == 0 {
		return ErrNotEnrolled
	}
	return nil
}

func (s *schoolService) ClassRoster(ctx context.Context, classID string) ([]domain.User, error) {
	if _, err := s.repo.FindClassByID(ctx, classID); err != nil {
		return nil, ErrClassNotFound
	}
	return s.repo.ClassMembers(ctx, classID)
}

func (s *schoolService) ClassMemberCount(ctx context.Context, classID string) (int64, error) {
	return s.repo.ClassMemberCount(ctx, classID)
}

func (s *schoolService) CreateAnnouncement(ctx context.Context, title, body, authorID string) (*domain.Announcement, error) {
	ann := &domain.Announcement{
		ID:        GenerateSecureID("ann"),
		Title:     title,
		Body:      body,
		AuthorID:  authorID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.CreateAnnouncement(ctx, ann); err != nil {
		return nil, err
	}
	return ann, nil
}

func (s *schoolService) ListAnnouncements(ctx context.Context, limit int) ([]domain.Announcement, error) {
	return s.repo.ListAnnouncements(ctx, limit)
}

func (s *schoolService) CreateAssignment(ctx context.Context, classID, title, description, activityID, createdBy, dueDate string) (*domain.Assignment, error) {
	class, err := s.repo.FindClassByID(ctx, classID)
	if err != nil {
		return nil, ErrClassNotFound
	}
	// Only the class's own teacher (or an admin) may set assignments.
	if class.TeacherID != createdBy {
		return nil, ErrNotClassTeacher
	}
	a := &domain.Assignment{
		ID:          GenerateSecureID("asg"),
		ClassID:     classID,
		Title:       title,
		Description: description,
		ActivityID:  activityID,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if dueDate != "" {
		parsed, err := time.Parse(time.RFC3339, dueDate)
		if err != nil {
			return nil, ErrInvalidDueDate
		}
		a.DueDate = parsed
	}
	if err := s.repo.CreateAssignment(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// AssignmentsForClass is the teacher view: only the class's own teacher may
// read its assignments and submissions (admins keep principal oversight).
func (s *schoolService) AssignmentsForClass(ctx context.Context, classID, callerID string, isAdmin bool) ([]domain.Assignment, error) {
	class, err := s.repo.FindClassByID(ctx, classID)
	if err != nil {
		return nil, ErrClassNotFound
	}
	if !isAdmin && class.TeacherID != callerID {
		return nil, ErrNotClassTeacher
	}
	return s.repo.AssignmentsForClass(ctx, classID)
}

func (s *schoolService) AssignmentsForLearner(ctx context.Context, learnerID string) ([]domain.Assignment, error) {
	return s.repo.AssignmentsForLearner(ctx, learnerID)
}

func (s *schoolService) SubmitAssignment(ctx context.Context, assignmentID, learnerID, note string) (*domain.Submission, error) {
	assignment, err := s.repo.FindAssignmentByID(ctx, assignmentID)
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	// Learners may only submit for classes they belong to.
	classes, err := s.repo.ClassesOfLearner(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, c := range classes {
		if c.ID == assignment.ClassID {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, ErrNotClassMember
	}
	sub := &domain.Submission{
		ID:           GenerateSecureID("sub"),
		AssignmentID: assignmentID,
		LearnerID:    learnerID,
		Note:         note,
		SubmittedAt:  time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.repo.SubmitAssignment(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *schoolService) FindSubmission(ctx context.Context, assignmentID, learnerID string) (*domain.Submission, error) {
	return s.repo.FindSubmission(ctx, assignmentID, learnerID)
}

// SubmissionsForAssignment is the teacher view: the caller must be the
// teacher of the assignment's class (learner submission notes are private to
// the class); admins keep oversight.
func (s *schoolService) SubmissionsForAssignment(ctx context.Context, assignmentID, callerID string, isAdmin bool) ([]domain.Submission, error) {
	assignment, err := s.repo.FindAssignmentByID(ctx, assignmentID)
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	class, err := s.repo.FindClassByID(ctx, assignment.ClassID)
	if err != nil {
		return nil, ErrClassNotFound
	}
	if !isAdmin && class.TeacherID != callerID {
		return nil, ErrNotClassTeacher
	}
	return s.repo.SubmissionsForAssignment(ctx, assignmentID)
}

func (s *schoolService) SubmissionCount(ctx context.Context, assignmentID string) (int64, error) {
	return s.repo.SubmissionCount(ctx, assignmentID)
}

func (s *schoolService) SubmissionCounts(ctx context.Context, assignmentIDs []string) (map[string]int64, error) {
	return s.repo.SubmissionCounts(ctx, assignmentIDs)
}

func (s *schoolService) WriteAuditLog(ctx context.Context, userID, action, detail, ip string) {
	entry := &domain.AuditLog{
		UserID:    userID,
		Action:    action,
		Detail:    detail,
		IP:        ip,
		CreatedAt: time.Now(),
	}
	if err := s.repo.WriteAuditLog(ctx, entry); err != nil {
		slog.Error("failed to write audit log", "action", action, "error", err)
	}
}

func (s *schoolService) ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AuditLog, int64, error) {
	return s.repo.ListAuditLogs(ctx, limit, offset)
}

func (s *schoolService) RevokeAll(ctx context.Context, userID string) error {
	return s.repo.RevokeAll(ctx, userID, time.Now())
}

func (s *schoolService) RevokedBefore(ctx context.Context, userID string) (*domain.UserRevocation, error) {
	return s.repo.RevokedBefore(ctx, userID)
}
