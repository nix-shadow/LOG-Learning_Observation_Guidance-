package service

import (
	"context"
	"errors"
	"time"

	"log-backend/internal/domain"
)

// Sentinel errors for the WP-2.1 parent portal. Handlers map these to honest
// HTTP statuses; anything else is a server error.
var (
	ErrParentInviteNotFound = errors.New("invite code not found or already used")
	ErrParentEmailTaken     = errors.New("an account with this email already exists")
	ErrParentScope          = errors.New("learner not linked to this parent")
	ErrInvalidDisclosure    = errors.New("parent portal consent requires disclosure_hash (sha256 hex of the notice text presented)")
)

// ParentChild is one learner visible to a parent — deliberately minimal:
// id + name + digest opt-in. No email, no phone, no contacts (WP-2.1 RC-04
// privacy boundary).
type ParentChild struct {
	StudentID   string `json:"id"`
	Name        string `json:"name"`
	DigestOptIn bool   `json:"digest_opt_in"`
}

// ParentLearner is the sanitized learner identity in a parent digest.
type ParentLearner struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ActivityDigest is the per-activity row in a parent digest — title, topic
// and the canonical supportive status, nothing else.
type ActivityDigest struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Topic  string `json:"topic"`
	Status string `json:"status"`
}

// ChildDigest is the read-only progress digest for one learner (WP-2.1
// RC-04). Rebuilt from the same GetDashboardData engine the learner sees —
// observations are deliberately excluded (teacher-internal), and no contact
// details are ever exposed.
type ChildDigest struct {
	Learner    ParentLearner     `json:"learner"`
	Progress   domain.Progress   `json:"progress"`
	Activities []ActivityDigest  `json:"activities"`
	Guidance   []domain.Guidance `json:"guidance"`
	AsOf       time.Time         `json:"as_of"`
}

type ParentService interface {
	// CreateParentInvite (teacher side): creates a pending parent link with a
	// one-time invite code for a student in the teacher's own classes. The
	// teacher's action is the school verification.
	CreateParentInvite(ctx context.Context, teacherID, studentID string) (*domain.ParentLink, error)
	// ParentSignup creates the PARENT account, claims the invite code, and
	// records the parent_access consent — atomically.
	ParentSignup(ctx context.Context, name, email, password, inviteCode, disclosureHash, language string) (*domain.User, string, error)
	LinkedChildren(ctx context.Context, parentID string) ([]ParentChild, error)
	ChildDigest(ctx context.Context, parentID, studentID string) (*ChildDigest, error)
	SetDigestOptIn(ctx context.Context, parentID, studentID string, enabled bool) error
}
