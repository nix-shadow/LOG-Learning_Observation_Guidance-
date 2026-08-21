package repository

import (
	"context"
	"time"

	"log-backend/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---------------------------------------------------------------------------
// WP-2.1 RC-04 — school-verified parent links
// ---------------------------------------------------------------------------

type parentRepo struct {
	db *gorm.DB
}

func NewParentRepository(db *gorm.DB) domain.ParentRepository {
	return &parentRepo{db: db}
}

func (r *parentRepo) CreateParentLink(ctx context.Context, link *domain.ParentLink) error {
	return r.db.WithContext(ctx).Create(link).Error
}

func (r *parentRepo) FindParentLinkByCode(ctx context.Context, code string) (*domain.ParentLink, error) {
	var link domain.ParentLink
	if err := r.db.WithContext(ctx).Where("invite_code = ?", code).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *parentRepo) FindLinkedParentLink(ctx context.Context, parentID, studentID string) (*domain.ParentLink, error) {
	var link domain.ParentLink
	if err := r.db.WithContext(ctx).
		Where("parent_id = ? AND student_id = ? AND status = ?", parentID, studentID, domain.ParentLinkStatusLinked).
		First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *parentRepo) LinkedChildren(ctx context.Context, parentID string) ([]domain.ParentLink, error) {
	var links []domain.ParentLink
	if err := r.db.WithContext(ctx).
		Where("parent_id = ? AND status = ?", parentID, domain.ParentLinkStatusLinked).
		Order("created_at").
		Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

func (r *parentRepo) UpdateParentLink(ctx context.Context, link *domain.ParentLink) error {
	return r.db.WithContext(ctx).Save(link).Error
}

// ClaimParentLinkTx creates the PARENT user, claims the pending link, and
// records the parent_access consent atomically. The claim is conditional on
// the link still being pending — a double-claim by two guardians can never
// both win.
func (r *parentRepo) ClaimParentLinkTx(ctx context.Context, user *domain.User, link *domain.ParentLink, consent *domain.ConsentRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		res := tx.Model(&domain.ParentLink{}).
			Where("id = ? AND status = ?", link.ID, domain.ParentLinkStatusPending).
			Updates(map[string]interface{}{
				"parent_id":     link.ParentID,
				"status":        domain.ParentLinkStatusLinked,
				"linked_at":     time.Now(),
				"digest_opt_in": link.DigestOptIn,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "consent_type"}},
			UpdateAll: true,
		}).Create(consent).Error
	})
}
