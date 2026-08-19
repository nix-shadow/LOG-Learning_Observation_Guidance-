package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"log-backend/internal/domain"
)

// PrivacyService owns consent recording, personal-data export, and account
// erasure (WP-0.1 of the phased implementation plan).
type PrivacyService interface {
	RecordConsent(ctx context.Context, userID string, in ConsentInput) (*domain.ConsentRecord, error)
	GetConsents(ctx context.Context, userID string) ([]domain.ConsentRecord, error)
	// HasActiveConsent feeds the server-side consent gate on learner mutation
	// routes (WP-0.1 enforcement round).
	HasActiveConsent(ctx context.Context, userID string) (bool, error)
	ExportData(ctx context.Context, userID string) (*domain.ExportBundle, error)
	DeleteAccount(ctx context.Context, userID string) error
	// ScrubDeletedData checkpoints (TRUNCATE) and VACUUMs the SQLite file so
	// logically-erased rows cannot be recovered from the WAL or freelist. The
	// report is the verification evidence for the operator log.
	ScrubDeletedData(ctx context.Context) (*domain.ScrubReport, error)
	// PurgeExpiredData enforces the retention schedule: learner accounts with
	// no activity for InactiveAccountRetentionYears are erased via the full
	// erasure map (so the school context stays consistent), and audit rows
	// older than AuditLogRetentionYears are deleted.
	PurgeExpiredData(ctx context.Context) (*PurgeReport, error)
}

// PurgeReport is the evidence for a retention-purge run: how many learner
// accounts were erased and how many audit rows were dropped.
type PurgeReport struct {
	UsersPurged     int64 `json:"users_purged"`
	AuditRowsPurged int64 `json:"audit_rows_purged"`
}

// ConsentInput is the validated consent payload. The handler binds and the
// service re-validates so no path (handler bug, future caller) can write an
// untyped consent row.
type ConsentInput struct {
	ConsentType     string
	Version         string
	GrantedBy       string
	GuardianName    string
	GuardianContact string
	Language        string
	Source          string
	DisclosureHash  string
	IP              string
}

// Sentinel errors for privacy operations.
var (
	ErrInvalidConsentType = errors.New("Invalid consent type. Must be guardian, terms, or privacy.")
	ErrInvalidGrantedBy   = errors.New("Invalid consent giver. Must be guardian, self, or school.")
	ErrInvalidLanguage    = errors.New("Invalid notice language. Must be ne or en.")
	ErrInvalidPolicy      = errors.New("Invalid policy version.")
)

// ConsentTypeValues mirrors the domain constants for validation without
// importing implementation details into the validation switch.
var consentTypeValues = map[string]bool{
	domain.ConsentTypeGuardian: true,
	domain.ConsentTypeTerms:    true,
	domain.ConsentTypePrivacy:  true,
}

var grantedByValues = map[string]bool{
	"guardian": true,
	"self":     true,
	"school":   true,
}

var languageValues = map[string]bool{"ne": true, "en": true}

// sha256HexOf normalizes a valid disclosure hash to its canonical form, or ""
// when it is not one. Accepted shapes (both lowercase hex):
//   - 64-char SHA-256 digest (WebCrypto available at consent time)
//   - "djb2-<hex>" — the documented fallback for non-secure contexts
//     (plain-HTTP school LANs without crypto.subtle); honestly labeled weaker
//     in the same spirit as the queue's enc:null fallback.
func sha256HexOf(s string) string {
	if len(s) == 64 {
		if _, err := hex.DecodeString(s); err == nil {
			return s
		}
	}
	if len(s) > 5 && s[:5] == "djb2-" {
		if _, err := hex.DecodeString(s[5:]); err == nil {
			return s
		}
	}
	return ""
}

type privacyService struct {
	repo domain.PrivacyRepository
}

func NewPrivacyService(repo domain.PrivacyRepository) PrivacyService {
	return &privacyService{repo: repo}
}

func (s *privacyService) RecordConsent(ctx context.Context, userID string, in ConsentInput) (*domain.ConsentRecord, error) {
	if !consentTypeValues[in.ConsentType] {
		return nil, ErrInvalidConsentType
	}
	if !grantedByValues[in.GrantedBy] {
		return nil, ErrInvalidGrantedBy
	}
	if !languageValues[in.Language] {
		return nil, ErrInvalidLanguage
	}
	if in.Version == "" {
		return nil, ErrInvalidPolicy
	}
	// WP-0.1 research (COPPA 16 CFR §312.5 practice): log the hash of the
	// exact disclosure text that was presented, so the school can later prove
	// what the guardian actually saw at consent time. Guardian consent without
	// it is rejected — an unprovable guardian grant is no evidence at all.
	if in.ConsentType == domain.ConsentTypeGuardian {
		if hash := sha256HexOf(in.DisclosureHash); hash == "" {
			return nil, errors.New("Guardian consent requires disclosure_hash (sha256 hex of the notice text presented).")
		} else {
			in.DisclosureHash = hash
		}
	}

	record := &domain.ConsentRecord{
		ID:              GenerateSecureID("csn"),
		UserID:          userID,
		ConsentType:     in.ConsentType,
		Version:         in.Version,
		Status:          domain.ConsentStatusGranted,
		GrantedBy:       in.GrantedBy,
		GuardianName:    in.GuardianName,
		GuardianContact: in.GuardianContact,
		Language:        in.Language,
		Source:          in.Source,
		DisclosureHash:  in.DisclosureHash,
		GrantedAt:       time.Now(),
		IP:              in.IP,
	}
	if err := s.repo.UpsertConsent(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *privacyService) GetConsents(ctx context.Context, userID string) ([]domain.ConsentRecord, error) {
	return s.repo.GetConsents(ctx, userID)
}

func (s *privacyService) HasActiveConsent(ctx context.Context, userID string) (bool, error) {
	return s.repo.HasActiveConsent(ctx, userID, domain.ConsentTypeGuardian)
}

// PurgeExpiredData runs the retention schedule. Learner erasure goes through
// DeleteAccount (the full erasure map + anonymized audit trail), never a bare
// DELETE — the school context must stay consistent even when the purge, not
// the learner, initiated the erasure.
func (s *privacyService) PurgeExpiredData(ctx context.Context) (*PurgeReport, error) {
	now := time.Now()
	inactiveCutoff := now.AddDate(-domain.InactiveAccountRetentionYears, 0, 0)
	auditCutoff := now.AddDate(-domain.AuditLogRetentionYears, 0, 0)

	ids, err := s.repo.FindInactiveStudentIDs(ctx, inactiveCutoff)
	if err != nil {
		return nil, err
	}
	report := &PurgeReport{}
	for _, id := range ids {
		if err := s.DeleteAccount(ctx, id); err != nil {
			// Fail loudly and stop: a partial purge that looks complete is
			// worse than a failed one the operator must investigate.
			return report, err
		}
		report.UsersPurged++
	}
	report.AuditRowsPurged, err = s.repo.DeleteOldAuditLogs(ctx, auditCutoff)
	if err != nil {
		return report, err
	}
	return report, nil
}

func (s *privacyService) ExportData(ctx context.Context, userID string) (*domain.ExportBundle, error) {
	bundle, err := s.repo.ExportData(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Never leak the password hash in an export intended for the user's own
	// eyes — it is the key to their account and useless to them as data.
	if bundle.User != nil {
		bundle.User.PasswordHash = ""
	}
	return bundle, nil
}

// DeleteAccount performs the erasure and writes the anonymized audit trail.
// The truncated SHA-256 of the user ID gives a joinable erasure trace without
// storing any personal reference (the ID itself is treated as personal data).
func (s *privacyService) DeleteAccount(ctx context.Context, userID string) error {
	sum := sha256.Sum256([]byte(userID))
	audit := &domain.AuditLog{
		UserID:    "",
		Action:    "privacy.account_deleted",
		Detail:    "erasure_hash=" + hex.EncodeToString(sum[:])[:16],
		IP:        "",
		CreatedAt: time.Now(),
	}
	return s.repo.DeleteAccountTx(ctx, userID, audit)
}

// ScrubDeletedData delegates to the repository's best-effort physical wipe
// and returns the verification report.
func (s *privacyService) ScrubDeletedData(ctx context.Context) (*domain.ScrubReport, error) {
	return s.repo.ScrubDeletedData(ctx)
}
