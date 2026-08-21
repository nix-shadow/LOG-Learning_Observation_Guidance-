package service

import (
	"context"
	"errors"

	"log-backend/internal/domain"
)

// Sentinel errors for admin operations (C3 seam: handlers map these to
// status codes; services own the business rules).
var (
	ErrInvalidRole  = errors.New("invalid role value. Must be STUDENT, MODERATOR, or ADMIN")
	ErrLastAdmin    = errors.New("last admin")
	ErrUserNotFound = errors.New("user not found")
)

// AdminService is the business layer behind the admin console (architecture
// review C3): dashboard analytics, user listing with guardian-consent status,
// role changes with the last-admin guard, and activity creation.
type AdminService interface {
	Dashboard(ctx context.Context) (*domain.SystemAnalytics, []domain.User, error)
	ListUsers(ctx context.Context, page, limit int) ([]domain.User, int64, map[string]domain.ConsentRecord, error)
	ChangeUserRole(ctx context.Context, actorID, targetID, ip string, role domain.Role) (*domain.User, error)
	CreateActivity(ctx context.Context, actorID, ip string, in CreateActivityInput) (*domain.Activity, error)
	// AnalyticsSummary (WP-4.3) is the opt-in-gated aggregate view.
	AnalyticsSummary(ctx context.Context) (domain.AnalyticsSummary, error)
}

// CreateActivityInput is a strict DTO that prevents clients from injecting
// server-managed fields like ID, CreatedAt, or Order.
type CreateActivityInput struct {
	Title         string
	Description   string
	Topic         string
	Difficulty    string
	Prerequisites string
	ContentJSON   string
}
