package service

import (
	"context"
	"errors"
	"testing"

	"log-backend/internal/domain"
)

// fakeAdminRepo records the role it was asked to write, so the service's
// role-validation rule is tested without a database.
type fakeAdminRepo struct {
	lastRole    domain.Role
	lastRoleSet bool
	err         error
}

func (f *fakeAdminRepo) DashboardStats(ctx context.Context) (int64, int64, int64, int64, []domain.User, error) {
	return 0, 0, 0, 0, nil, nil
}
func (f *fakeAdminRepo) ListUsers(ctx context.Context, page, limit int) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeAdminRepo) GuardianConsentMap(ctx context.Context, userIDs []string) (map[string]domain.ConsentRecord, error) {
	return map[string]domain.ConsentRecord{}, nil
}
func (f *fakeAdminRepo) ChangeRoleTx(ctx context.Context, targetID string, role domain.Role, actorID, ip string) (*domain.User, error) {
	f.lastRole = role
	f.lastRoleSet = true
	return &domain.User{ID: targetID, Role: role}, f.err
}
func (f *fakeAdminRepo) CreateActivity(ctx context.Context, act *domain.Activity, actorID, ip string) error {
	return nil
}
func (f *fakeAdminRepo) AnalyticsSummary(ctx context.Context) (domain.AnalyticsSummary, error) {
	return domain.AnalyticsSummary{TotalUsers: 2, OptedInUsers: 1, Completions: 1}, nil
}

// TestAnalyticsSummaryDelegatesAndIsNeverFabricated (WP-4.3): the service
// passes the repo's opt-in-gated aggregates through untouched — including
// the honest nil AvgScore when no opted-in learner has completions.
func TestAnalyticsSummaryDelegatesAndIsNeverFabricated(t *testing.T) {
	svc := NewAdminService(&fakeAdminRepo{})
	sum, err := svc.AnalyticsSummary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.TotalUsers != 2 || sum.OptedInUsers != 1 || sum.Completions != 1 {
		t.Fatalf("summary not delegated: %+v", sum)
	}
	if sum.AvgScore != nil {
		t.Fatalf("AvgScore must stay nil when the repo reports none, got %v", *sum.AvgScore)
	}
}

// TestChangeUserRoleRejectsInvalidRoles pins the C3 role check: the service
// owns the rule that only the three defined role constants are accepted —
// any arbitrary string (e.g. "SUPERUSER") is rejected before touching the
// repository.
func TestChangeUserRoleRejectsInvalidRoles(t *testing.T) {
	repo := &fakeAdminRepo{}
	svc := NewAdminService(repo)

	for _, bad := range []domain.Role{"SUPERUSER", "admin", "", " PRINCIPAL "} {
		if _, err := svc.ChangeUserRole(context.Background(), "actor", "target", "", bad); !errors.Is(err, ErrInvalidRole) {
			t.Fatalf("role %q: expected ErrInvalidRole, got %v", bad, err)
		}
		if repo.lastRoleSet {
			t.Fatalf("role %q: repository must not be reached for an invalid role", bad)
		}
	}
}

func TestChangeUserRoleAcceptsDefinedRoles(t *testing.T) {
	for _, good := range []domain.Role{domain.RoleStudent, domain.RoleModerator, domain.RoleAdmin} {
		repo := &fakeAdminRepo{}
		svc := NewAdminService(repo)
		user, err := svc.ChangeUserRole(context.Background(), "actor", "target", "", good)
		if err != nil {
			t.Fatalf("role %q: unexpected error %v", good, err)
		}
		if user.Role != good || !repo.lastRoleSet || repo.lastRole != good {
			t.Fatalf("role %q: service did not delegate correctly (repo role %q, set=%v)", good, repo.lastRole, repo.lastRoleSet)
		}
	}
}

// TestChangeUserRolePropagatesRepoErrors ensures sentinel errors from the
// transaction (last-admin guard, not-found) pass straight through for the
// handler to map.
func TestChangeUserRolePropagatesRepoErrors(t *testing.T) {
	svc := NewAdminService(&fakeAdminRepo{err: ErrLastAdmin})
	if _, err := svc.ChangeUserRole(context.Background(), "actor", "admin-1", "", domain.RoleModerator); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin to propagate, got %v", err)
	}
}
