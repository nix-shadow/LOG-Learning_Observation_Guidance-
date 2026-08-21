package repository

import (
	"context"
	"time"

	"log-backend/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type schoolRepo struct {
	db *gorm.DB
}

func NewSchoolRepository(db *gorm.DB) domain.SchoolRepository {
	return &schoolRepo{db: db}
}

func (r *schoolRepo) CreateClass(ctx context.Context, class *domain.Class) error {
	return r.db.WithContext(ctx).Create(class).Error
}

func (r *schoolRepo) FindClassByID(ctx context.Context, id string) (*domain.Class, error) {
	var class domain.Class
	if err := r.db.WithContext(ctx).First(&class, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &class, nil
}

// FindClassByInviteCode resolves the class behind a join code (WP-1.5).
func (r *schoolRepo) FindClassByInviteCode(ctx context.Context, code string) (*domain.Class, error) {
	var class domain.Class
	if err := r.db.WithContext(ctx).First(&class, "invite_code = ?", code).Error; err != nil {
		return nil, err
	}
	return &class, nil
}

func (r *schoolRepo) ListClasses(ctx context.Context) ([]domain.Class, error) {
	var classes []domain.Class
	if err := r.db.WithContext(ctx).Order("grade, section").Find(&classes).Error; err != nil {
		return nil, err
	}
	return classes, nil
}

func (r *schoolRepo) ListClassesByTeacher(ctx context.Context, teacherID string) ([]domain.Class, error) {
	var classes []domain.Class
	if err := r.db.WithContext(ctx).Where("teacher_id = ?", teacherID).Order("grade, section").Find(&classes).Error; err != nil {
		return nil, err
	}
	return classes, nil
}

func (r *schoolRepo) Enroll(ctx context.Context, classID string, userIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, uid := range userIDs {
			// Validate the target user is a STUDENT (never enroll staff by accident)
			var count int64
			if err := tx.Model(&domain.User{}).Where("id = ? AND role = ?", uid, domain.RoleStudent).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				continue
			}
			member := domain.ClassMember{ClassID: classID, UserID: uid, JoinedAt: time.Now()}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *schoolRepo) RemoveMember(ctx context.Context, classID, userID string) (int64, error) {
	res := r.db.WithContext(ctx).Where("class_id = ? AND user_id = ?", classID, userID).Delete(&domain.ClassMember{})
	return res.RowsAffected, res.Error
}

func (r *schoolRepo) ClassMembers(ctx context.Context, classID string) ([]domain.User, error) {
	var users []domain.User
	err := r.db.WithContext(ctx).
		Joins("JOIN class_members ON class_members.user_id = users.id").
		Where("class_members.class_id = ?", classID).
		Order("users.name").
		Find(&users).Error
	return users, err
}

func (r *schoolRepo) ClassMemberCount(ctx context.Context, classID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.ClassMember{}).Where("class_id = ?", classID).Count(&count).Error
	return count, err
}

func (r *schoolRepo) ClassesOfLearner(ctx context.Context, learnerID string) ([]domain.Class, error) {
	var classes []domain.Class
	err := r.db.WithContext(ctx).
		Joins("JOIN class_members ON class_members.class_id = classes.id").
		Where("class_members.user_id = ?", learnerID).
		Order("classes.grade, classes.section").
		Find(&classes).Error
	return classes, err
}

// StudentInTeacherClasses is the WP-1.5 per-student progress scope check: the
// learner must be a member of at least one class the teacher owns. Honest
// false on any miss — a teacher can never read a stranger's progress.
func (r *schoolRepo) StudentInTeacherClasses(ctx context.Context, teacherID, learnerID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.ClassMember{}).
		Joins("JOIN classes ON classes.id = class_members.class_id").
		Where("class_members.user_id = ? AND classes.teacher_id = ?", learnerID, teacherID).
		Count(&count).Error
	return count > 0, err
}

// FindUserByEmail / CreateStudentUser back the roster CSV import (WP-1.5).
func (r *schoolRepo) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *schoolRepo) CreateStudentUser(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *schoolRepo) CreateAnnouncement(ctx context.Context, ann *domain.Announcement) error {
	return r.db.WithContext(ctx).Create(ann).Error
}

func (r *schoolRepo) ListAnnouncements(ctx context.Context, limit int) ([]domain.Announcement, error) {
	var anns []domain.Announcement
	if err := r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Find(&anns).Error; err != nil {
		return nil, err
	}
	return anns, nil
}

func (r *schoolRepo) CreateAssignment(ctx context.Context, a *domain.Assignment) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *schoolRepo) FindAssignmentByID(ctx context.Context, id string) (*domain.Assignment, error) {
	var a domain.Assignment
	if err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *schoolRepo) AssignmentsForClass(ctx context.Context, classID string) ([]domain.Assignment, error) {
	var as []domain.Assignment
	if err := r.db.WithContext(ctx).Where("class_id = ?", classID).Order("due_date asc").Find(&as).Error; err != nil {
		return nil, err
	}
	return as, nil
}

func (r *schoolRepo) AssignmentsForLearner(ctx context.Context, learnerID string) ([]domain.Assignment, error) {
	var as []domain.Assignment
	err := r.db.WithContext(ctx).
		Joins("JOIN class_members ON class_members.class_id = assignments.class_id").
		Where("class_members.user_id = ?", learnerID).
		Order("assignments.due_date asc").
		Find(&as).Error
	return as, err
}

func (r *schoolRepo) SubmitAssignment(ctx context.Context, s *domain.Submission) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "assignment_id"}, {Name: "learner_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"note", "updated_at"}),
	}).Create(s).Error
}

func (r *schoolRepo) FindSubmission(ctx context.Context, assignmentID, learnerID string) (*domain.Submission, error) {
	var sub domain.Submission
	if err := r.db.WithContext(ctx).
		Where("assignment_id = ? AND learner_id = ?", assignmentID, learnerID).
		First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *schoolRepo) SubmissionsForAssignment(ctx context.Context, assignmentID string) ([]domain.Submission, error) {
	var subs []domain.Submission
	if err := r.db.WithContext(ctx).Where("assignment_id = ?", assignmentID).Order("submitted_at asc").Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

func (r *schoolRepo) SubmissionCount(ctx context.Context, assignmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Submission{}).Where("assignment_id = ?", assignmentID).Count(&count).Error
	return count, err
}

// SubmissionCounts returns submission counts for many assignments in one
// batched query (GROUP BY) so the teacher's assignment list never issues a
// COUNT per assignment.
func (r *schoolRepo) SubmissionCounts(ctx context.Context, assignmentIDs []string) (map[string]int64, error) {
	counts := make(map[string]int64)
	if len(assignmentIDs) == 0 {
		return counts, nil
	}
	type row struct {
		AssignmentID string
		Count        int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Model(&domain.Submission{}).
		Select("assignment_id, COUNT(*) AS count").
		Where("assignment_id IN ?", assignmentIDs).
		Group("assignment_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, rw := range rows {
		counts[rw.AssignmentID] = rw.Count
	}
	return counts, nil
}

func (r *schoolRepo) WriteAuditLog(ctx context.Context, entry *domain.AuditLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *schoolRepo) ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AuditLog, int64, error) {
	var entries []domain.AuditLog
	var total int64
	r.db.WithContext(ctx).Model(&domain.AuditLog{}).Count(&total)
	if err := r.db.WithContext(ctx).Order("id desc").Limit(limit).Offset(offset).Find(&entries).Error; err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func (r *schoolRepo) RevokeAll(ctx context.Context, userID string, before time.Time) error {
	rec := domain.UserRevocation{UserID: userID, RevokedBefore: before}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"revoked_before"}),
	}).Create(&rec).Error
}

func (r *schoolRepo) RevokedBefore(ctx context.Context, userID string) (*domain.UserRevocation, error) {
	var rec domain.UserRevocation
	if err := r.db.WithContext(ctx).First(&rec, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}
