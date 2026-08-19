package service

import (
	"context"
	"errors"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

// ErrActivityNotFound marks a completion attempt for an activity that does not
// exist. Handlers map ONLY this sentinel to 404 — every other completion
// failure is a server error, so a transient DB issue never looks like "activity
// not found" (which would make the offline queue delete the learner's work).
var ErrActivityNotFound = errors.New("Activity not found")

// ErrCourseNotFound maps to 404 on enroll/unenroll. Anything else is a 500 —
// a failed enroll must never look like a missing course.
var ErrCourseNotFound = errors.New("Course not found")

type ActivityResponse struct {
	domain.Activity
	Status string `json:"status"`
}

type LearnerService interface {
	GetDashboardData(ctx context.Context, learnerID string) (domain.User, domain.Progress, []ActivityResponse, []domain.Observation, []domain.Guidance, error)
	GetLearningJourneyData(ctx context.Context, learnerID string) ([]ActivityResponse, error)
	GetChartData(ctx context.Context, learnerID string) ([]map[string]interface{}, error)
	CompleteActivity(ctx context.Context, learnerID, activityID string, stats domain.AttemptStats) (domain.Observation, domain.Guidance, error)
}

type CourseService interface {
	GetCourses(ctx context.Context, userID string, page, limit int) ([]domain.Course, int64, error)
	GetMicroModules(ctx context.Context, activityID string) ([]domain.MicroModule, error)
	Enroll(ctx context.Context, userID, courseID string) error
	Unenroll(ctx context.Context, userID, courseID string) error
}

type ModeratorService interface {
	GetModeratorRoster(ctx context.Context, callerID string, page, limit int) ([]map[string]interface{}, int64, int64, int64, string, error)
}

type learnerService struct {
	userRepo        domain.UserRepository
	activityRepo    domain.ActivityRepository
	progressRepo    domain.ProgressRepository
	learnerDataRepo domain.LearnerDataRepository
	completionRepo  domain.CompletionRepository
}

func NewLearnerService(u domain.UserRepository, a domain.ActivityRepository, p domain.ProgressRepository, l domain.LearnerDataRepository, c domain.CompletionRepository) LearnerService {
	return &learnerService{userRepo: u, activityRepo: a, progressRepo: p, learnerDataRepo: l, completionRepo: c}
}

func (s *learnerService) GetDashboardData(ctx context.Context, learnerID string) (domain.User, domain.Progress, []ActivityResponse, []domain.Observation, []domain.Guidance, error) {
	user, err := s.userRepo.FindByID(ctx, learnerID)
	if err != nil {
		return domain.User{}, domain.Progress{}, nil, nil, nil, err
	}

	dbActivities, _ := s.activityRepo.FindAll(ctx)
	learnerActs, _ := s.progressRepo.FindLearnerActivities(ctx, learnerID)

	statusMap := make(map[string]string)
	for _, la := range learnerActs {
		statusMap[la.ActivityID] = la.Status
	}

	var activities []ActivityResponse
	for _, act := range dbActivities {
		status := "Pending"
		if st, ok := statusMap[act.ID]; ok {
			status = st
		}
		activities = append(activities, ActivityResponse{
			Activity: act,
			Status:   status,
		})
	}

	progress, err := s.progressRepo.FindProgress(ctx, learnerID)
	if err != nil || progress == nil {
		progress = &domain.Progress{
			LearnerID:     learnerID,
			TotalTopics:   len(activities),
			Completed:     0,
			CurrentStreak: 0,
			OverallScore:  0,
		}
		s.progressRepo.SaveProgress(ctx, progress)
	}

	observations, _ := s.learnerDataRepo.FindObservations(ctx, learnerID)
	guidance, _ := s.learnerDataRepo.FindGuidance(ctx, learnerID)

	return *user, *progress, activities, observations, guidance, nil
}

func (s *learnerService) GetLearningJourneyData(ctx context.Context, learnerID string) ([]ActivityResponse, error) {
	dbActivities, _ := s.activityRepo.FindAll(ctx)
	learnerActs, _ := s.progressRepo.FindLearnerActivities(ctx, learnerID)

	statusMap := make(map[string]string)
	for _, la := range learnerActs {
		statusMap[la.ActivityID] = la.Status
	}

	var activities []ActivityResponse
	for _, act := range dbActivities {
		status := "Pending"
		if st, ok := statusMap[act.ID]; ok {
			status = st
		}
		activities = append(activities, ActivityResponse{
			Activity: act,
			Status:   status,
		})
	}
	return activities, nil
}

// GetChartData returns ONLY real DailyActivity rows. Research round (WP-0.2):
// the old fallback fabricated a 7-day all-zero series for learners with no
// activity — invented numbers, violating AGENTS.md §1. The honest empty
// response is an empty list; the frontend chart renders a real empty state.
func (s *learnerService) GetChartData(ctx context.Context, learnerID string) ([]map[string]interface{}, error) {
	acts, _ := s.learnerDataRepo.FindDailyActivities(ctx, learnerID)
	chartData := make([]map[string]interface{}, 0, len(acts))
	for _, act := range acts {
		chartData = append(chartData, map[string]interface{}{
			"name":     act.DayName,
			"score":    act.Score,
			"duration": act.Duration,
		})
	}
	return chartData, nil
}

func (s *learnerService) CompleteActivity(ctx context.Context, learnerID, activityID string, stats domain.AttemptStats) (domain.Observation, domain.Guidance, error) {
	// Delegated to the completion repository, which runs the full flow
	// (learner activity, progress, observation, guidance) in one transaction.
	return s.completionRepo.CompleteActivityTx(ctx, learnerID, activityID, stats)
}

type courseService struct {
	courseRepo domain.CourseRepository
}

func NewCourseService(c domain.CourseRepository) CourseService {
	return &courseService{courseRepo: c}
}

func (s *courseService) GetCourses(ctx context.Context, userID string, page, limit int) ([]domain.Course, int64, error) {
	return s.courseRepo.FindCourses(ctx, userID, page, limit)
}

func (s *courseService) Enroll(ctx context.Context, userID, courseID string) error {
	if err := s.courseRepo.Enroll(ctx, userID, courseID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCourseNotFound
		}
		return err
	}
	return nil
}

func (s *courseService) Unenroll(ctx context.Context, userID, courseID string) error {
	return s.courseRepo.Unenroll(ctx, userID, courseID)
}

func (s *courseService) GetMicroModules(ctx context.Context, activityID string) ([]domain.MicroModule, error) {
	return s.courseRepo.FindModulesByActivityID(ctx, activityID)
}

type moderatorService struct {
	modRepo domain.ModeratorRepository
}

func NewModeratorService(m domain.ModeratorRepository) ModeratorService {
	return &moderatorService{modRepo: m}
}

// GetModeratorRoster returns the roster scoped to the caller's own classes,
// with every stat derived from real backend data:
//   - total = students enrolled in the caller's classes (never school-wide)
//   - assignmentsDue = real assignments in the caller's classes whose due date
//     has passed (hardcoded 0 was a fabricated number — AGENTS.md §1)
//   - className = the caller's first real class name, "" when they have none
func (s *moderatorService) GetModeratorRoster(ctx context.Context, callerID string, page, limit int) ([]map[string]interface{}, int64, int64, int64, string, error) {
	rosterData, total, needsAttention, err := s.modRepo.GetRoster(ctx, callerID, page, limit)
	if err != nil {
		return nil, 0, 0, 0, "", err
	}

	var roster []map[string]interface{}
	for _, entry := range rosterData {
		progress := entry.Progress

		completion := 0
		if progress.TotalTopics > 0 {
			completion = int(float64(progress.Completed) / float64(progress.TotalTopics) * 100)
		}

		status := "Active"
		if progress.CurrentStreak == 0 {
			status = "Inactive"
		}

		lastActive := "—"
		if !entry.User.UpdatedAt.IsZero() {
			lastActive = entry.User.UpdatedAt.Format("Jan 02")
		}

		roster = append(roster, map[string]interface{}{
			"id":          entry.User.ID,
			"name":        entry.User.Name,
			"completion":  completion,
			"streak":      progress.CurrentStreak,
			"status":      status,
			"last_active": lastActive,
		})
	}

	assignmentsDue, err := s.modRepo.AssignmentsDueForTeacher(ctx, callerID)
	if err != nil {
		assignmentsDue = 0
	}
	className, err := s.modRepo.FirstClassNameForTeacher(ctx, callerID)
	if err != nil {
		className = ""
	}

	return roster, total, needsAttention, assignmentsDue, className, nil
}
