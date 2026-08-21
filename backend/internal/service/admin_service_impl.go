package service

import (
	"context"
	"time"

	"log-backend/internal/domain"
)

type adminService struct {
	repo domain.AdminRepository
}

func NewAdminService(repo domain.AdminRepository) AdminService {
	return &adminService{repo: repo}
}

func (s *adminService) Dashboard(ctx context.Context) (*domain.SystemAnalytics, []domain.User, error) {
	totalUsers, _, totalCompletions, activeDaily, recentUsers, err := s.repo.DashboardStats(ctx)
	if err != nil {
		return nil, nil, err
	}
	return &domain.SystemAnalytics{
		TotalUsers:       int(totalUsers),
		ActiveDaily:      int(activeDaily),
		TotalCompletions: int(totalCompletions),
	}, recentUsers, nil
}

func (s *adminService) ListUsers(ctx context.Context, page, limit int) ([]domain.User, int64, map[string]domain.ConsentRecord, error) {
	users, total, err := s.repo.ListUsers(ctx, page, limit)
	if err != nil {
		return nil, 0, nil, err
	}
	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	consent, err := s.repo.GuardianConsentMap(ctx, userIDs)
	if err != nil {
		return nil, 0, nil, err
	}
	return users, total, consent, nil
}

// ChangeUserRole validates the role constant (rejecting any arbitrary
// string) and delegates the check-then-act guard + audit to the repository's
// single transaction.
func (s *adminService) ChangeUserRole(ctx context.Context, actorID, targetID, ip string, role domain.Role) (*domain.User, error) {
	switch role {
	case domain.RoleStudent, domain.RoleModerator, domain.RoleAdmin:
		// valid — proceed
	default:
		return nil, ErrInvalidRole
	}
	return s.repo.ChangeRoleTx(ctx, targetID, role, actorID, ip)
}

// CreateActivity auto-assigns the display order and persists with its audit
// entry atomically.
func (s *adminService) CreateActivity(ctx context.Context, actorID, ip string, in CreateActivityInput) (*domain.Activity, error) {
	act := &domain.Activity{
		ID:            GenerateSecureID("act"), // Server-generated ID
		Title:         in.Title,
		Description:   in.Description,
		Topic:         in.Topic,
		Difficulty:    in.Difficulty,
		Prerequisites: in.Prerequisites,
		ContentJSON:   in.ContentJSON,
		CreatedAt:     time.Now(),
	}
	// Assign the next display order inside the same transaction as the insert
	// so two concurrent creates cannot collide on one slot.
	if err := s.repo.CreateActivity(ctx, act, actorID, ip); err != nil {
		return nil, err
	}
	return act, nil
}

// AnalyticsSummary delegates to the repo's opt-in-gated aggregates (WP-4.3).
func (s *adminService) AnalyticsSummary(ctx context.Context) (domain.AnalyticsSummary, error) {
	return s.repo.AnalyticsSummary(ctx)
}
