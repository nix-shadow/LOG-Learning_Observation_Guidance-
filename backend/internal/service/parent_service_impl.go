package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type parentService struct {
	parentRepo     domain.ParentRepository
	userRepo       domain.UserRepository
	schoolRepo     domain.SchoolRepository
	learnerService LearnerService
}

func NewParentService(
	parentRepo domain.ParentRepository,
	userRepo domain.UserRepository,
	schoolRepo domain.SchoolRepository,
	learnerService LearnerService,
) ParentService {
	return &parentService{
		parentRepo:     parentRepo,
		userRepo:       userRepo,
		schoolRepo:     schoolRepo,
		learnerService: learnerService,
	}
}

// CreateParentInvite (WP-2.1): the teacher's action IS the school
// verification. Scope is enforced like per-student progress: a teacher can
// only invite a parent for a learner in their own classes (hard 404).
func (s *parentService) CreateParentInvite(ctx context.Context, teacherID, studentID string) (*domain.ParentLink, error) {
	scoped, err := s.schoolRepo.StudentInTeacherClasses(ctx, teacherID, studentID)
	if err != nil {
		return nil, err
	}
	if !scoped {
		return nil, ErrNotClassTeacher
	}

	code := GenerateInviteCode()
	for i := 0; i < 5; i++ {
		if _, err := s.parentRepo.FindParentLinkByCode(ctx, code); err != nil {
			break
		}
		code = GenerateInviteCode()
	}

	link := &domain.ParentLink{
		ID:         GenerateSecureID("plink"),
		StudentID:  studentID,
		InviteCode: code,
		Status:     domain.ParentLinkStatusPending,
		CreatedBy:  teacherID,
		CreatedAt:  time.Now(),
	}
	if err := s.parentRepo.CreateParentLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

// ParentSignup (WP-2.1): creates the PARENT account, claims the pending
// invite, and records the parent_access consent — all-or-nothing. The
// disclosure hash is required and validated exactly like guardian consent:
// a parent-portal grant without evidence of the notice shown is no grant.
func (s *parentService) ParentSignup(ctx context.Context, name, email, password, inviteCode, disclosureHash, language string) (*domain.User, string, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	inviteCode = strings.ToUpper(strings.TrimSpace(inviteCode))
	if name == "" || email == "" || password == "" {
		return nil, "", errors.New("name, email and password are required")
	}
	if len(password) < 8 {
		return nil, "", errors.New("password must be at least 8 characters")
	}

	if _, err := s.userRepo.FindByEmail(ctx, email); err == nil {
		return nil, "", ErrParentEmailTaken
	}

	link, err := s.parentRepo.FindParentLinkByCode(ctx, inviteCode)
	if err != nil || link.Status != domain.ParentLinkStatusPending {
		return nil, "", ErrParentInviteNotFound
	}

	hash := sha256HexOf(disclosureHash)
	if hash == "" {
		return nil, "", ErrInvalidDisclosure
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, "", err
	}

	user := &domain.User{
		ID:           GenerateSecureID("user-p"),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         domain.RoleParent,
		IsVerified:   true,
		CreatedAt:    time.Now(),
	}
	now := time.Now()
	claimed := *link
	claimed.ParentID = user.ID
	claimed.Status = domain.ParentLinkStatusLinked
	claimed.LinkedAt = &now

	consent := &domain.ConsentRecord{
		ID:             GenerateSecureID("csn"),
		UserID:         user.ID,
		ConsentType:    domain.ConsentTypeParentAccess,
		Version:        domain.PolicyVersion,
		Status:         domain.ConsentStatusGranted,
		GrantedBy:      "guardian",
		GuardianName:   name,
		Language:       language,
		Source:         "parent_signup",
		DisclosureHash: hash,
		GrantedAt:      now,
	}

	if err := s.parentRepo.ClaimParentLinkTx(ctx, user, &claimed, consent); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrParentInviteNotFound
		}
		return nil, "", err
	}

	token, err := GenerateJWT(user.ID, user.Role)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *parentService) LinkedChildren(ctx context.Context, parentID string) ([]ParentChild, error) {
	links, err := s.parentRepo.LinkedChildren(ctx, parentID)
	if err != nil {
		return nil, err
	}
	children := make([]ParentChild, 0, len(links))
	for _, l := range links {
		student, err := s.userRepo.FindByID(ctx, l.StudentID)
		if err != nil {
			continue
		}
		children = append(children, ParentChild{
			StudentID:   l.StudentID,
			Name:        student.Name,
			DigestOptIn: l.DigestOptIn,
		})
	}
	return children, nil
}

// ChildDigest (WP-2.1): read-only, sanitized, honest. Built on the WP-1.1
// status engine (GetDashboardData), with observations excluded and the
// learner identity reduced to id + name — no emails, no phones, no OTPs.
func (s *parentService) ChildDigest(ctx context.Context, parentID, studentID string) (*ChildDigest, error) {
	link, err := s.parentRepo.FindLinkedParentLink(ctx, parentID, studentID)
	if err != nil {
		return nil, ErrParentScope
	}

	user, progress, activities, _, guidance, err := s.learnerService.GetDashboardData(ctx, studentID)
	if err != nil {
		return nil, err
	}

	digest := &ChildDigest{
		Learner:  ParentLearner{ID: user.ID, Name: user.Name},
		Progress: progress,
		AsOf:     time.Now().UTC(),
	}
	for _, a := range activities {
		digest.Activities = append(digest.Activities, ActivityDigest{
			ID:     a.ID,
			Title:  a.Title,
			Topic:  a.Topic,
			Status: a.Status,
		})
	}
	digest.Guidance = guidance
	_ = link
	return digest, nil
}

func (s *parentService) SetDigestOptIn(ctx context.Context, parentID, studentID string, enabled bool) error {
	link, err := s.parentRepo.FindLinkedParentLink(ctx, parentID, studentID)
	if err != nil {
		return ErrParentScope
	}
	link.DigestOptIn = enabled
	return s.parentRepo.UpdateParentLink(ctx, link)
}
