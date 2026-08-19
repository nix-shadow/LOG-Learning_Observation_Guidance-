package repository

import (
	"context"
	"errors"
	"time"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type privacyRepo struct {
	db *gorm.DB
}

func NewPrivacyRepository(db *gorm.DB) domain.PrivacyRepository {
	return &privacyRepo{db: db}
}

func (r *privacyRepo) UpsertConsent(ctx context.Context, record *domain.ConsentRecord) error {
	var existing domain.ConsentRecord
	err := r.db.WithContext(ctx).First(&existing, "user_id = ? AND consent_type = ?", record.UserID, record.ConsentType).Error
	if err == nil {
		existing.Version = record.Version
		existing.Status = record.Status
		existing.GrantedBy = record.GrantedBy
		existing.GuardianName = record.GuardianName
		existing.GuardianContact = record.GuardianContact
		existing.Language = record.Language
		existing.Source = record.Source
		existing.GrantedAt = record.GrantedAt
		existing.WithdrawnAt = record.WithdrawnAt
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *privacyRepo) GetConsents(ctx context.Context, userID string) ([]domain.ConsentRecord, error) {
	var records []domain.ConsentRecord
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("granted_at asc").Find(&records).Error
	return records, err
}

func (r *privacyRepo) GetConsentMap(ctx context.Context, userIDs []string, consentType string) (map[string]domain.ConsentRecord, error) {
	result := make(map[string]domain.ConsentRecord, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	var records []domain.ConsentRecord
	err := r.db.WithContext(ctx).
		Where("user_id IN ? AND consent_type = ? AND status = ?", userIDs, consentType, domain.ConsentStatusGranted).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	for _, rec := range records {
		// Keep the most recent grant per user (rows are ordered by granted_at
		// within the map builder, so a later row simply overwrites).
		prev, ok := result[rec.UserID]
		if !ok || rec.GrantedAt.After(prev.GrantedAt) {
			result[rec.UserID] = rec
		}
	}
	return result, nil
}

// HasActiveConsent powers the server-side consent gate (research round): a
// "granted" row for the (user, type) pair that was never withdrawn. A
// withdrawal flips the same row, so this is a simple status check — there is
// no history to replay.
func (r *privacyRepo) HasActiveConsent(ctx context.Context, userID, consentType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.ConsentRecord{}).
		Where("user_id = ? AND consent_type = ? AND status = ?", userID, consentType, domain.ConsentStatusGranted).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindInactiveStudentIDs returns learner-role users whose last activity
// predates the cutoff. "Last activity" is the most recent of the user's own
// progress row or learner-activity row — a learner who has not produced any
// data since the cutoff is a retention candidate. Staff accounts are never
// retention candidates.
func (r *privacyRepo) FindInactiveStudentIDs(ctx context.Context, olderThan time.Time) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ?", domain.RoleStudent).
		Where("updated_at < ?", olderThan).
		Where("id NOT IN (SELECT learner_id FROM learner_activities WHERE completed_at >= ?)", olderThan).
		Where("id NOT IN (SELECT learner_id FROM progresses WHERE last_activity_date >= ?)", olderThan).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// DeleteOldAuditLogs removes audit rows older than the retention window
// (3 years, AuditLogRetentionYears). Returns the number of rows removed.
func (r *privacyRepo) DeleteOldAuditLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("created_at < ?", olderThan).Delete(&domain.AuditLog{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *privacyRepo) ExportData(ctx context.Context, userID string) (*domain.ExportBundle, error) {
	bundle := &domain.ExportBundle{}

	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	bundle.User = &user

	if records, err := r.GetConsents(ctx, userID); err == nil {
		bundle.Consents = records
	}

	if err := r.db.WithContext(ctx).Where("learner_id = ?", userID).Order("completed_at asc").Find(&bundle.LearnerActivities).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).First(&bundle.Progress, "learner_id = ?", userID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if err := r.db.WithContext(ctx).Where("learner_id = ?", userID).Order("created_at asc").Find(&bundle.Observations).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("learner_id = ?", userID).Order("created_at asc").Find(&bundle.Guidance).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("learner_id = ?", userID).Order("date asc").Find(&bundle.DailyActivities).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Joins("JOIN class_members ON class_members.class_id = classes.id AND class_members.user_id = ?", userID).Find(&bundle.Classes).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("learner_id = ?", userID).Order("submitted_at desc").Find(&bundle.Submissions).Error; err != nil {
		return nil, err
	}
	// The user's own audit trail — the actions they performed (limited view).
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Limit(200).Find(&bundle.AuditLog).Error; err != nil {
		return nil, err
	}
	return bundle, nil
}

// DeleteAccountTx implements the erasure data map. No FK constraints exist in
// this schema (gorm.AutoMigrate does not create them), so every table that can
// reference a user is handled explicitly, child tables first:
//
//	DELETE   — learner-private rows and auth plumbing
//	ANONYMIZE — rows that must survive for others' context (audit trail,
//	            announcements, assignments, classes), user reference blanked
//	DELETE   — the user row itself (hard delete, Unscoped)
func (r *privacyRepo) DeleteAccountTx(ctx context.Context, userID string, audit *domain.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user domain.User
		if err := tx.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
			return err
		}

		// Learner-private data — deleted outright.
		if err := tx.Where("learner_id = ?", userID).Delete(&domain.LearnerActivity{}).Error; err != nil {
			return err
		}
		if err := tx.Where("learner_id = ?", userID).Delete(&domain.Progress{}).Error; err != nil {
			return err
		}
		if err := tx.Where("learner_id = ?", userID).Delete(&domain.Observation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("learner_id = ?", userID).Delete(&domain.Guidance{}).Error; err != nil {
			return err
		}
		if err := tx.Where("learner_id = ?", userID).Delete(&domain.DailyActivity{}).Error; err != nil {
			return err
		}
		if err := tx.Where("learner_id = ?", userID).Delete(&domain.Submission{}).Error; err != nil {
			return err
		}
		// Membership rows — the user leaves every class.
		if err := tx.Where("user_id = ?", userID).Delete(&domain.ClassMember{}).Error; err != nil {
			return err
		}
		// Auth plumbing — sessions die with the account.
		if err := tx.Where("user_id = ?", userID).Delete(&domain.TokenBlocklist{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&domain.UserRevocation{}).Error; err != nil {
			return err
		}
		// Consent evidence — the record is the user's own data; it is erased
		// with the account (the anonymized audit entry below is the trail).
		if err := tx.Where("user_id = ?", userID).Delete(&domain.ConsentRecord{}).Error; err != nil {
			return err
		}
		if user.Phone != nil {
			if err := tx.Where("phone = ?", *user.Phone).Delete(&domain.OTPRecord{}).Error; err != nil {
				return err
			}
		}

		// Rows that must survive for others' context — anonymize the reference.
		if err := tx.Model(&domain.AuditLog{}).Where("user_id = ?", userID).Update("user_id", "").Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Announcement{}).Where("author_id = ?", userID).Update("author_id", "").Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Assignment{}).Where("created_by = ?", userID).Update("created_by", "").Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Class{}).Where("teacher_id = ?", userID).Update("teacher_id", "").Error; err != nil {
			return err
		}

		// The erasure trail — written atomically with the erasure, UserID
		// blanked so no personal reference survives. The Detail carries a
		// truncated SHA-256 of the erased user ID for cross-referencing
		// without storing personal data.
		if audit != nil {
			audit.UserID = ""
			audit.CreatedAt = time.Now()
			if err := tx.Create(audit).Error; err != nil {
				return err
			}
		}

		return tx.Unscoped().Delete(&domain.User{}, "id = ?", userID).Error
	})
}

// ScrubDeletedData physically wipes logically-deleted rows. Research finding
// (SQLite forensics): a plain DELETE leaves recoverable cells in freelist
// pages and the WAL file — secure_delete=ON (set in the DSN) zeroes b-tree
// cells, but the WAL keeps old page versions until checkpoint+truncate, and
// freelist pages survive until VACUUM. Order matters: checkpoint first
// (per sqlite.org forum workaround for chunk-size truncation), then VACUUM.
// Best-effort by contract — the erasure already committed; callers log errors.
// The returned report is the verification evidence (freelist counts + WAL
// frames) so the handler can log that the wipe actually shrank the surface.
func (r *privacyRepo) ScrubDeletedData(ctx context.Context) (*domain.ScrubReport, error) {
	report := &domain.ScrubReport{}

	// Before: how many free pages could still hold erased rows.
	if err := r.db.WithContext(ctx).Raw("PRAGMA freelist_count").Scan(&report.FreelistBefore).Error; err != nil {
		return report, err
	}

	// Checkpoint(TRUNCATE) returns (busy, log, checkpointed): "log" is the
	// WAL size in frames before truncation — the recoverable pages folded away.
	var checkpoint struct {
		Busy         int
		Log          int
		Checkpointed int
	}
	if err := r.db.WithContext(ctx).Raw("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&checkpoint).Error; err != nil {
		return report, err
	}
	report.WALFrames = int64(checkpoint.Log)
	report.WALBytes = report.WALFrames * 4096 // SQLite default page size

	// VACUUM rebuilds the file, dropping freelist pages entirely.
	if err := r.db.WithContext(ctx).Exec("VACUUM").Error; err != nil {
		return report, err
	}

	// After: the surviving freelist (normally 0 after a full VACUUM).
	if err := r.db.WithContext(ctx).Raw("PRAGMA freelist_count").Scan(&report.FreelistAfter).Error; err != nil {
		return report, err
	}
	return report, nil
}
