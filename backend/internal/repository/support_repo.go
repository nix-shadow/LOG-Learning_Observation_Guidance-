package repository

import (
	"context"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// WP-2.2 RC-06 — support issues (who-to-call funnel)
// ---------------------------------------------------------------------------

type supportRepo struct {
	db *gorm.DB
}

func NewSupportRepository(db *gorm.DB) domain.SupportRepository {
	return &supportRepo{db: db}
}

func (r *supportRepo) CreateIssue(ctx context.Context, issue *domain.SupportIssue) error {
	return r.db.WithContext(ctx).Create(issue).Error
}

func (r *supportRepo) IssuesByUser(ctx context.Context, userID string) ([]domain.SupportIssue, error) {
	var issues []domain.SupportIssue
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

func (r *supportRepo) OpenEscalatedIssues(ctx context.Context) ([]domain.SupportIssue, error) {
	var issues []domain.SupportIssue
	if err := r.db.WithContext(ctx).
		Where("escalated = ? AND status = ?", true, domain.SupportStatusOpen).
		Order("created_at ASC").
		Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

func (r *supportRepo) FindIssueByID(ctx context.Context, id string) (*domain.SupportIssue, error) {
	var issue domain.SupportIssue
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&issue).Error; err != nil {
		return nil, err
	}
	return &issue, nil
}

func (r *supportRepo) ResolveIssue(ctx context.Context, issue *domain.SupportIssue) error {
	return r.db.WithContext(ctx).Save(issue).Error
}

// ---------------------------------------------------------------------------
// WP-2.3 RC-08 — teacher annotations on learners
// ---------------------------------------------------------------------------

type noteRepo struct {
	db *gorm.DB
}

func NewNoteRepository(db *gorm.DB) domain.NoteRepository {
	return &noteRepo{db: db}
}

func (r *noteRepo) FindNote(ctx context.Context, studentID string) (*domain.LearnerNote, error) {
	var note domain.LearnerNote
	if err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Order("updated_at DESC").First(&note).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *noteRepo) UpsertNote(ctx context.Context, note *domain.LearnerNote) error {
	return r.db.WithContext(ctx).Save(note).Error
}
