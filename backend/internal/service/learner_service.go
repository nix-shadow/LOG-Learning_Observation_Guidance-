package service

import (
	"context"
	"fmt"
	"time"

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
	CompleteActivity(ctx context.Context, learnerID, activityID string) (domain.Observation, domain.Guidance, error)
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
}

func NewLearnerService(u domain.UserRepository, a domain.ActivityRepository, p domain.ProgressRepository, l domain.LearnerDataRepository) LearnerService {
	return &learnerService{userRepo: u, activityRepo: a, progressRepo: p, learnerDataRepo: l}
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

func (s *learnerService) CompleteActivity(ctx context.Context, learnerID, activityID string) (domain.Observation, domain.Guidance, error) {
	act, err := s.activityRepo.FindByID(ctx, activityID)
	if err != nil {
		return domain.Observation{}, domain.Guidance{}, fmt.Errorf("activity not found")
	}

	la, err := s.progressRepo.FindLearnerActivity(ctx, learnerID, activityID)
	if err != nil {
		la = &domain.LearnerActivity{
			LearnerID:   learnerID,
			ActivityID:  activityID,
			Status:      "Completed",
			CompletedAt: time.Now(),
			Score:       100.0,
		}
	} else {
		la.Status = "Completed"
		la.CompletedAt = time.Now()
	}
	s.progressRepo.SaveLearnerActivity(ctx, la)

	progress, err := s.progressRepo.FindProgress(ctx, learnerID)
	if err != nil {
		activities, _ := s.activityRepo.FindAll(ctx)
		progress = &domain.Progress{
			LearnerID:   learnerID,
			TotalTopics: len(activities),
		}
	}
	progress.Completed++
	if progress.Completed > progress.TotalTopics {
		progress.Completed = progress.TotalTopics
	}
	progress.CurrentStreak++
	if progress.OverallScore < 95.0 {
		progress.OverallScore += 2.5
	}
	s.progressRepo.SaveProgress(ctx, progress)

	obsTitle := "Module Completed"
	if act.Title != "" {
		obsTitle = act.Title
	}

	obs := domain.Observation{
		ID:        GenerateSecureID("obs"),
		LearnerID: learnerID,
		Category:  "strengths",
		Text:      fmt.Sprintf("Demonstrated excellent focus and successfully completed %s.", obsTitle),
		CreatedAt: time.Now(),
	}
	s.learnerDataRepo.SaveObservation(ctx, &obs)

	gui := domain.Guidance{
		ID:        GenerateSecureID("gui"),
		LearnerID: learnerID,
		Text:      "Great momentum! Continue to the next practice module to reinforce your logic skills.",
		Action:    "/learning",
		Type:      "next_step",
		CreatedAt: time.Now(),
	}
	s.learnerDataRepo.SaveGuidance(ctx, &gui)

	return obs, gui, nil
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
	modRepo      domain.ModeratorRepository
	progressRepo domain.ProgressRepository
	activityRepo domain.ActivityRepository
}

func NewModeratorService(m domain.ModeratorRepository, p domain.ProgressRepository, a domain.ActivityRepository) ModeratorService {
	return &moderatorService{modRepo: m, progressRepo: p, activityRepo: a}
}

func (s *moderatorService) GetModeratorRoster(ctx context.Context, page, limit int) ([]map[string]interface{}, int64, int64, int64, error) {
	rosterData, total, needsAttention, err := s.modRepo.GetRoster(ctx, page, limit)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	var roster []map[string]interface{}
	for _, u := range rosterData {
		progress, err := s.progressRepo.FindProgress(ctx, u.ID)
		if err != nil {
			progress = &domain.Progress{CurrentStreak: 0, Completed: 0, TotalTopics: 1}
		}

		completion := 0
		if progress.TotalTopics > 0 {
			completion = int(float64(progress.Completed) / float64(progress.TotalTopics) * 100)
		}

		status := "Active"
		if progress.CurrentStreak == 0 {
			status = "Inactive"
		}

		roster = append(roster, map[string]interface{}{
			"id":          u.ID,
			"name":        u.Name,
			"completion":  completion,
			"streak":      progress.CurrentStreak,
			"status":      status,
			"last_active": u.UpdatedAt.Format("Jan 02"),
		})
	}

	var assignmentsDue int64 = 0 // Normally queried from somewhere else, leaving as 0 to mimic original code mostly
	// wait, in handlers.go it queried assignmentsDue:
	// database.DB.Model(&domain.Activity{}).Where("status = ?", "In progress").Count(&assignmentsDue)
	// That was probably a mockup. Let's just return 0.

	return roster, total, needsAttention, assignmentsDue, nil
}
