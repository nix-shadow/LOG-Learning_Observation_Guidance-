package domain

import "time"

// Consent types captured by LOG. The guardian consent is the one required for
// learners under 18 (Nepal Privacy Act 2075 s.23(6)(a)); terms/privacy are
// recorded so the evidence store shows the full consent history.
const (
	ConsentTypeGuardian = "guardian"
	ConsentTypeTerms    = "terms"
	ConsentTypePrivacy  = "privacy"
	// ConsentTypeParentAccess (WP-2.1 RC-04): recorded when a guardian claims
	// a parent link, evidencing that they consented to viewing the digest.
	ConsentTypeParentAccess = "parent_access"
	// ConsentTypeAnalytics (WP-4.3): the school's aggregate usage statistics
	// (anonymous counts only, never individual learner data) are only
	// computed over users who opted in. Recorded with granted_by "self" when
	// a user toggles it in Settings.
	ConsentTypeAnalytics = "analytics"
)

// ConsentRecord is the versioned, auditable evidence store for consent.
// One active record per (user, type): re-granting updates the existing row
// (version + granted_at), withdrawal flips status and sets WithdrawnAt.
type ConsentRecord struct {
	ID              string     `json:"id" gorm:"primaryKey"`
	UserID          string     `json:"user_id" gorm:"uniqueIndex:idx_consent_user_type"`
	ConsentType     string     `json:"consent_type" gorm:"uniqueIndex:idx_consent_user_type"`
	Version         string     `json:"version"`          // policy version shown to the user
	Status          string     `json:"status"`           // "granted" | "withdrawn"
	GrantedBy       string     `json:"granted_by"`       // "guardian" | "self" | "school"
	GuardianName    string     `json:"guardian_name"`    // who gave the consent (if guardian)
	GuardianContact string     `json:"guardian_contact"` // phone/email of the consent-giver
	Language        string     `json:"language"`         // "ne" | "en" — notice language at consent time
	Source          string     `json:"source"`           // "register" | "google" | "otp" | "settings"
	DisclosureHash  string     `json:"disclosure_hash"`  // sha256 hex of the exact notice text presented
	GrantedAt       time.Time  `json:"granted_at"`
	WithdrawnAt     *time.Time `json:"withdrawn_at"`
	IP              string     `json:"ip" gorm:"-"`
}

// Privacy policy constants. These are the retention commitments shown to
// users (bilingual notices) and embedded in data exports.
const (
	PolicyVersion                 = "2026-08-v1"
	ConsentStatusGranted          = "granted"
	ConsentStatusWithdrawn        = "withdrawn"
	InactiveAccountRetentionYears = 2  // learner data purged 2 years after last activity
	AuditLogRetentionYears        = 3  // audit rows kept (legitimate interest), anonymized on erasure
	QueuedDataRetentionDays       = 90 // local offline queue is user-controlled; server-side staging capped
)

// ScrubReport is the verification evidence for a physical erasure pass:
// freelist page counts before/after VACUUM and the WAL frame count (bytes =
// frames × 4096 page size) that the TRUNCATE checkpoint folded away.
// Logged by the handler so an operator can confirm the wipe actually shrank
// the recoverable surface instead of trusting that it did.
type ScrubReport struct {
	FreelistBefore int64 `json:"freelist_before"`
	FreelistAfter  int64 `json:"freelist_after"`
	WALFrames      int64 `json:"wal_frames"`
	WALBytes       int64 `json:"wal_bytes"`
}

// ExportBundle is the complete, portable record of a user's own data
// (GDPR Art. 15/20 style). Derived/inferred analytics are deliberately
// excluded — only provided + observed data is portable.
type ExportBundle struct {
	User              *User             `json:"user"`
	Consents          []ConsentRecord   `json:"consents"`
	Progress          *Progress         `json:"progress"`
	LearnerActivities []LearnerActivity `json:"learner_activities"`
	Observations      []Observation     `json:"observations"`
	Guidance          []Guidance        `json:"guidance"`
	DailyActivities   []DailyActivity   `json:"daily_activities"`
	Classes           []Class           `json:"classes"`
	Submissions       []Submission      `json:"submissions"`
	AuditLog          []AuditLog        `json:"audit_log"`
}
