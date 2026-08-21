package service

import (
	"context"
	"errors"
	"time"

	"log-backend/internal/domain"
)

// Sentinel errors for the WP-2.2 support funnel.
var (
	ErrIssueNotFound = errors.New("issue not found")
	ErrBadCategory   = errors.New("unknown issue category")
)

// SupportCategories are the fixed funnel categories; each maps to a bilingual
// guidance article in the frontend (support namespace) plus the escalation
// path into the moderator inbox.
var SupportCategories = map[string]bool{
	"device":       true,
	"connectivity": true,
	"account":      true,
	"content":      true,
	"other":        true,
}

type SupportService interface {
	CreateIssue(ctx context.Context, userID, category, description string, escalated bool) (*domain.SupportIssue, error)
	MyIssues(ctx context.Context, userID string) ([]domain.SupportIssue, error)
	Inbox(ctx context.Context) ([]domain.SupportIssue, error)
	ResolveIssue(ctx context.Context, issueID, resolverID, note string) (*domain.SupportIssue, error)
}

type supportService struct {
	repo domain.SupportRepository
}

func NewSupportService(repo domain.SupportRepository) SupportService {
	return &supportService{repo: repo}
}

func (s *supportService) CreateIssue(ctx context.Context, userID, category, description string, escalated bool) (*domain.SupportIssue, error) {
	if !SupportCategories[category] {
		return nil, ErrBadCategory
	}
	issue := &domain.SupportIssue{
		ID:          GenerateSecureID("iss"),
		UserID:      userID,
		Category:    category,
		Description: description,
		Escalated:   escalated,
		Status:      domain.SupportStatusOpen,
		CreatedAt:   time.Now(),
	}
	if err := s.repo.CreateIssue(ctx, issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *supportService) MyIssues(ctx context.Context, userID string) ([]domain.SupportIssue, error) {
	return s.repo.IssuesByUser(ctx, userID)
}

// Inbox returns only OPEN, ESCALATED issues — the ones a moderator or admin
// must act on. Self-served issues (solved by guidance) never clutter it.
func (s *supportService) Inbox(ctx context.Context) ([]domain.SupportIssue, error) {
	return s.repo.OpenEscalatedIssues(ctx)
}

func (s *supportService) ResolveIssue(ctx context.Context, issueID, resolverID, note string) (*domain.SupportIssue, error) {
	issue, err := s.repo.FindIssueByID(ctx, issueID)
	if err != nil {
		return nil, ErrIssueNotFound
	}
	if issue.Status == domain.SupportStatusResolved {
		return issue, nil // idempotent — resolving twice is a no-op
	}
	now := time.Now()
	issue.Status = domain.SupportStatusResolved
	issue.ResolverID = resolverID
	issue.ResolutionNote = note
	issue.ResolvedAt = &now
	if err := s.repo.ResolveIssue(ctx, issue); err != nil {
		return nil, err
	}
	return issue, nil
}
