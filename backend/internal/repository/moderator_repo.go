package repository

import (
	"context"
	"time"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type moderatorRepo struct {
	db *gorm.DB
}

func NewModeratorRepository(db *gorm.DB) domain.ModeratorRepository {
	return &moderatorRepo{db: db}
}

// teacherScope returns a join chain that narrows every roster query to the
// caller's own classes: a teacher must never see another teacher's students
// (the old shape counted every STUDENT in the school).
func (r *moderatorRepo) teacherScope(q *gorm.DB, teacherID string) *gorm.DB {
	return q.
		Joins("JOIN class_members ON class_members.user_id = users.id").
		Joins("JOIN classes ON classes.id = class_members.class_id").
		Where("classes.teacher_id = ?", teacherID)
}

// GetRoster returns one page of the caller's students paired with their
// progress using a constant number of queries (roster page + one batched
// progress lookup), regardless of roster size.
func (r *moderatorRepo) GetRoster(ctx context.Context, teacherID string, page, limit int) ([]domain.RosterEntry, int64, int64, error) {
	var total int64
	var needsAttention int64

	if err := r.teacherScope(r.db.WithContext(ctx).Model(&domain.User{}), teacherID).
		Where("users.role = ?", domain.RoleStudent).
		Distinct("users.id").
		Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	// Students in the caller's classes whose streak has lapsed.
	if err := r.db.WithContext(ctx).
		Model(&domain.Progress{}).
		Joins("JOIN users ON users.id = progresses.learner_id").
		Joins("JOIN class_members ON class_members.user_id = users.id").
		Joins("JOIN classes ON classes.id = class_members.class_id").
		Where("classes.teacher_id = ?", teacherID).
		Where("users.role = ?", domain.RoleStudent).
		Where("progresses.current_streak = 0").
		Distinct("progresses.learner_id").
		Count(&needsAttention).Error; err != nil {
		return nil, 0, 0, err
	}

	offset := (page - 1) * limit
	var roster []domain.User
	if err := r.teacherScope(r.db.WithContext(ctx).Model(&domain.User{}), teacherID).
		Where("users.role = ?", domain.RoleStudent).
		Distinct("users.id").
		Order("users.name").
		Offset(offset).
		Limit(limit).
		Find(&roster).Error; err != nil {
		return nil, 0, 0, err
	}

	// Single batched progress lookup for the whole page (no per-learner loop).
	learnerIDs := make([]string, 0, len(roster))
	for _, u := range roster {
		learnerIDs = append(learnerIDs, u.ID)
	}
	progressByID := map[string]domain.Progress{}
	if len(learnerIDs) > 0 {
		var progresses []domain.Progress
		if err := r.db.WithContext(ctx).
			Where("learner_id IN ?", learnerIDs).
			Find(&progresses).Error; err != nil {
			return nil, 0, 0, err
		}
		for _, p := range progresses {
			progressByID[p.LearnerID] = p
		}
	}

	entries := make([]domain.RosterEntry, 0, len(roster))
	for _, u := range roster {
		entries = append(entries, domain.RosterEntry{User: u, Progress: progressByID[u.ID]})
	}
	return entries, total, needsAttention, nil
}

// AssignmentsDueForTeacher counts assignments in the caller's classes whose
// due date has passed — real data, not a hardcoded zero.
func (r *moderatorRepo) AssignmentsDueForTeacher(ctx context.Context, teacherID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Assignment{}).
		Joins("JOIN classes ON classes.id = assignments.class_id").
		Where("classes.teacher_id = ?", teacherID).
		Where("assignments.due_date IS NOT NULL AND assignments.due_date <> ?", time.Time{}).
		Where("assignments.due_date < ?", time.Now()).
		Count(&count).Error
	return count, err
}

// FirstClassNameForTeacher returns the caller's first class name (stable
// order) or "" when they teach no classes — the moderator overview derives
// its label from this instead of a course title.
func (r *moderatorRepo) FirstClassNameForTeacher(ctx context.Context, teacherID string) (string, error) {
	var name string
	err := r.db.WithContext(ctx).
		Model(&domain.Class{}).
		Where("teacher_id = ?", teacherID).
		Order("grade, section").
		Limit(1).
		Pluck("name", &name).Error
	if err != nil {
		return "", err
	}
	return name, nil
}
