package domain

import "time"

// RoleParent is a guardian account (WP-2.1 RC-04). Parents see a read-only
// digest of their linked learners' progress — they never take learner
// actions, so the consent gate (which governs learner mutations) does not
// apply to them.
const RoleParent Role = "PARENT"

// ParentLink is a school-verified guardian→learner link (WP-2.1 RC-04).
// A teacher creates a pending link with a one-time invite code; the guardian
// claims it at signup, which flips status to "linked". The teacher's action
// IS the school verification — no fabricated phone/SMS check.
type ParentLink struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	ParentID    string     `json:"parent_id" gorm:"index"`
	StudentID   string     `json:"student_id" gorm:"index"`
	InviteCode  string     `json:"invite_code" gorm:"uniqueIndex"`
	Status      string     `json:"status" gorm:"index"`
	CreatedBy   string     `json:"created_by"` // teacher user ID who verified
	DigestOptIn bool       `json:"digest_opt_in"`
	CreatedAt   time.Time  `json:"created_at"`
	LinkedAt    *time.Time `json:"linked_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
}

const (
	ParentLinkStatusPending = "pending"
	ParentLinkStatusLinked  = "linked"
	ParentLinkStatusRevoked = "revoked"
)

// SupportIssue is one entry in the who-to-call support funnel (WP-2.2 RC-06):
// the reporter picks a category, sees the matching guidance, and either marks
// it solved or escalates to the school's moderator/admin inbox.
type SupportIssue struct {
	ID             string     `json:"id" gorm:"primaryKey"`
	UserID         string     `json:"user_id" gorm:"index"`
	Category       string     `json:"category"` // device | connectivity | account | content | other
	Description    string     `json:"description"`
	Escalated      bool       `json:"escalated"`
	Status         string     `json:"status" gorm:"index"` // open | resolved
	ResolverID     string     `json:"resolver_id"`
	ResolutionNote string     `json:"resolution_note"`
	CreatedAt      time.Time  `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
}

const (
	SupportStatusOpen     = "open"
	SupportStatusResolved = "resolved"
)

// LearnerNote is a teacher's supportive annotation on a learner (WP-2.3
// RC-08). One editable note per learner; the template chips in the UI are
// phrased supportively ("This area could use more practice").
type LearnerNote struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	StudentID string    `json:"student_id" gorm:"index"`
	TeacherID string    `json:"teacher_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
