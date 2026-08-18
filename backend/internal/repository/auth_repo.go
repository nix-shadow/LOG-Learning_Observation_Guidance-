package repository

import (
	"context"
	"log-backend/internal/domain"
	"time"

	"gorm.io/gorm"
)

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) domain.AuthRepository {
	return &authRepo{db: db}
}

func (r *authRepo) SaveOTP(ctx context.Context, record *domain.OTPRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

func (r *authRepo) FindOTPByPhone(ctx context.Context, phone string) (*domain.OTPRecord, error) {
	var record domain.OTPRecord
	if err := r.db.WithContext(ctx).First(&record, "phone = ?", phone).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *authRepo) IncrementOTPAttempts(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Model(&domain.OTPRecord{}).Where("phone = ?", phone).UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *authRepo) DeleteOTP(ctx context.Context, phone string) error {
	// Delete ALL OTPs for this phone — a verified or invalidated OTP must not
	// remain usable for replay within its expiry window.
	return r.db.WithContext(ctx).Where("phone = ?", phone).Delete(&domain.OTPRecord{}).Error
}

func (r *authRepo) DeleteExpiredOTPs(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", time.Now().Add(-10*time.Minute)).Delete(&domain.OTPRecord{}).Error
}

func (r *authRepo) BlockToken(ctx context.Context, blocklist *domain.TokenBlocklist) error {
	return r.db.WithContext(ctx).Save(blocklist).Error
}

func (r *authRepo) IsTokenBlocked(ctx context.Context, jti string) (bool, error) {
	var blocked domain.TokenBlocklist
	err := r.db.WithContext(ctx).First(&blocked, "jti = ?", jti).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
