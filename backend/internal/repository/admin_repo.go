package repository

import (
	"context"
	"database/sql"
	"time"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"gorm.io/gorm"
)

type adminRepo struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) domain.AdminRepository {
	return &adminRepo{db: db}
}

// DashboardStats aggregates the admin console numbers. Active daily = learners
// with real completion activity in the last 24 hours. users.updated_at only
// moves on password/role changes, so counting users by it made this metric
// permanently ~0 — LearnerActivity.completed_at is the honest signal.
func (r *adminRepo) DashboardStats(ctx context.Context) (totalUsers, totalActivities, totalCompletions, activeDaily int64, recentUsers []domain.User, err error) {
	db := r.db.WithContext(ctx)
	if err = db.Model(&domain.User{}).Count(&totalUsers).Error; err != nil {
		return 0, 0, 0, 0, nil, err
	}
	if err = db.Model(&domain.Activity{}).Count(&totalActivities).Error; err != nil {
		return 0, 0, 0, 0, nil, err
	}
	if err = db.Model(&domain.Progress{}).Select("COALESCE(SUM(completed), 0)").Scan(&totalCompletions).Error; err != nil {
		return 0, 0, 0, 0, nil, err
	}
	if err = db.Model(&domain.LearnerActivity{}).
		Where("completed_at > ?", time.Now().Add(-24*time.Hour)).
		Distinct("learner_id").
		Count(&activeDaily).Error; err != nil {
		return 0, 0, 0, 0, nil, err
	}
	if err = db.Order("created_at desc").Limit(5).Find(&recentUsers).Error; err != nil {
		return 0, 0, 0, 0, nil, err
	}
	return totalUsers, totalActivities, totalCompletions, activeDaily, recentUsers, nil
}

// ListUsers pages deterministically (created_at desc — SQLite's default
// rowid order made the first page arbitrary, and the admin UI's enrollment
// form relies on this list).
func (r *adminRepo) ListUsers(ctx context.Context, page, limit int) ([]domain.User, int64, error) {
	db := r.db.WithContext(ctx)
	var users []domain.User
	var total int64
	if err := db.Model(&domain.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// GuardianConsentMap returns the most recent active guardian consent per
// user. Additive field — absent users are simply missing from the map.
func (r *adminRepo) GuardianConsentMap(ctx context.Context, userIDs []string) (map[string]domain.ConsentRecord, error) {
	if len(userIDs) == 0 {
		return map[string]domain.ConsentRecord{}, nil
	}
	var consentRows []domain.ConsentRecord
	if err := r.db.WithContext(ctx).
		Where("user_id IN ? AND consent_type = ? AND status = ?", userIDs, domain.ConsentTypeGuardian, domain.ConsentStatusGranted).
		Find(&consentRows).Error; err != nil {
		return nil, err
	}
	consentByUser := make(map[string]domain.ConsentRecord, len(consentRows))
	for _, rec := range consentRows {
		if prev, ok := consentByUser[rec.UserID]; !ok || rec.GrantedAt.After(prev.GrantedAt) {
			consentByUser[rec.UserID] = rec
		}
	}
	return consentByUser, nil
}

// ChangeRoleTx changes a user's role. The check-then-act last-admin guard,
// the role write, and the audit entry run inside ONE transaction so two
// concurrent demotions can never leave a school with zero admins, and a
// failed write never reports success. Returns the updated user.
func (r *adminRepo) ChangeRoleTx(ctx context.Context, targetID string, role domain.Role, actorID, ip string) (*domain.User, error) {
	var updated *domain.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user domain.User
		if err := tx.First(&user, "id = ?", targetID).Error; err != nil {
			return service.ErrUserNotFound
		}

		// Never demote the last remaining ADMIN — a school with zero principals
		// has no recovery path (nobody can promote a new one). Re-counted inside
		// the transaction so a concurrent demotion cannot pass the guard.
		if user.Role == domain.RoleAdmin && role != domain.RoleAdmin {
			var adminCount int64
			if err := tx.Model(&domain.User{}).
				Where("role = ? AND id <> ?", domain.RoleAdmin, targetID).
				Count(&adminCount).Error; err != nil {
				return err
			}
			if adminCount < 1 {
				return service.ErrLastAdmin
			}
		}

		user.Role = role
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// Append-only audit trail for sensitive privilege changes — written in
		// the same transaction as the mutation, so the trail cannot drift.
		if err := tx.Create(&domain.AuditLog{
			UserID:    actorID,
			Action:    "user.role_change",
			Detail:    targetID + " -> " + string(role),
			IP:        ip,
			CreatedAt: time.Now(),
		}).Error; err != nil {
			return err
		}
		updated = &user
		return nil
	})
	return updated, err
}

// CreateActivity persists the activity and its audit entry in one
// transaction. The DTO layer already guards server-managed fields; the
// display order is assigned inside the same transaction so two concurrent
// creates cannot collide on one slot.
func (r *adminRepo) CreateActivity(ctx context.Context, act *domain.Activity, actorID, ip string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&domain.Activity{}).Count(&count).Error; err != nil {
			return err
		}
		act.Order = int(count) + 1
		if err := tx.Create(act).Error; err != nil {
			return err
		}
		return tx.Create(&domain.AuditLog{
			UserID:    actorID,
			Action:    "activity.create",
			Detail:    act.ID + " " + act.Title,
			IP:        ip,
			CreatedAt: time.Now(),
		}).Error
	})
}

// AnalyticsSummary (WP-4.3) computes aggregates ONLY over learners with an
// active "analytics" consent record. The opt-in set is read first; every
// aggregate query is constrained to that set, so a school cannot see usage
// numbers for learners who did not consent. Zero opted-in learners yields
// zero/absent values — never invented rows.
func (r *adminRepo) AnalyticsSummary(ctx context.Context) (domain.AnalyticsSummary, error) {
	db := r.db.WithContext(ctx)
	var sum domain.AnalyticsSummary

	if err := db.Model(&domain.User{}).Count(&sum.TotalUsers).Error; err != nil {
		return sum, err
	}

	var optedIn []string
	if err := db.Model(&domain.ConsentRecord{}).
		Where("consent_type = ? AND status = ?", domain.ConsentTypeAnalytics, domain.ConsentStatusGranted).
		Pluck("user_id", &optedIn).Error; err != nil {
		return sum, err
	}
	sum.OptedInUsers = int64(len(optedIn))
	if len(optedIn) == 0 {
		return sum, nil
	}

	if err := db.Model(&domain.LearnerActivity{}).
		Where("learner_id IN ? AND status = ?", optedIn, domain.StatusCompleted).
		Count(&sum.Completions).Error; err != nil {
		return sum, err
	}
	if err := db.Model(&domain.LearnerActivity{}).
		Where("learner_id IN ? AND completed_at > ?", optedIn, time.Now().Add(-24*time.Hour)).
		Distinct("learner_id").
		Count(&sum.ActiveDaily).Error; err != nil {
		return sum, err
	}

	var avg sql.NullFloat64
	if err := db.Model(&domain.LearnerActivity{}).
		Where("learner_id IN ? AND status = ?", optedIn, domain.StatusCompleted).
		Select("AVG(score)").
		Scan(&avg).Error; err != nil {
		return sum, err
	}
	if avg.Valid {
		sum.AvgScore = &avg.Float64
	}
	return sum, nil
}
