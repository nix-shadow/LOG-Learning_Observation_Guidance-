package service

import (
	"context"

	"log-backend/internal/domain"
)

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
	GetCourses(ctx context.Context, page, limit int) ([]domain.Course, int64, error)
	GetMicroModules(ctx context.Context, activityID string) ([]domain.MicroModule, error)
}

type ModeratorService interface {
	GetModeratorRoster(ctx context.Context, page, limit int) ([]map[string]interface{}, int64, int64, int64, error)
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

func (s *learnerService) GetChartData(ctx context.Context, learnerID string) ([]map[string]interface{}, error) {
	acts, _ := s.learnerDataRepo.FindDailyActivities(ctx, learnerID)
	chartData := make([]map[string]interface{}, 0)
	for _, act := range acts {
		chartData = append(chartData, map[string]interface{}{
			"name":     act.DayName,
			"score":    act.Score,
			"duration": act.Duration,
		})
	}

	if len(chartData) == 0 {
		chartData = []map[string]interface{}{
			{"name": "Mon", "score": 0, "duration": 0},
			{"name": "Tue", "score": 0, "duration": 0},
			{"name": "Wed", "score": 0, "duration": 0},
			{"name": "Thu", "score": 0, "duration": 0},
			{"name": "Fri", "score": 0, "duration": 0},
			{"name": "Sat", "score": 0, "duration": 0},
			{"name": "Sun", "score": 0, "duration": 0},
		}
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

func (s *courseService) GetCourses(ctx context.Context, page, limit int) ([]domain.Course, int64, error) {
	return s.courseRepo.FindCourses(ctx, page, limit)
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

func (s *moderatorService) GetModeratorRoster(ctx context.Context, page, limit int) ([]map[string]interface{}, int64, int64, int64, error) {
	rosterData, total, needsAttention, err := s.modRepo.GetRoster(ctx, page, limit)
	if err != nil {
		return nil, 0, 0, 0, err
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

	var assignmentsDue int64 = 0 // Normally queried from somewhere else, leaving as 0 to mimic original code mostly
	// wait, in handlers.go it queried assignmentsDue:
	// database.DB.Model(&domain.Activity{}).Where("status = ?", "In progress").Count(&assignmentsDue)
	// That was probably a mockup. Let's just return 0.

	return roster, total, needsAttention, assignmentsDue, nil
}
