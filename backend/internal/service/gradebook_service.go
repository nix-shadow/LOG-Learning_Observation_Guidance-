package service

import (
	"context"
	"errors"
	"time"

	"log-backend/internal/domain"
)

// Sentinel errors for the WP-2.3 honest gradebook.
var (
	ErrClassNotFoundForGradebook = errors.New("class not found or not yours")
)

// GradebookRow is one real data point: the learner's canonical status plus
// the REAL stored accuracy and attempt count for an activity. When a learner
// has no rows at all the frontend renders the honest "Not yet assessed"
// state — never an invented grade.
type GradebookRow struct {
	ActivityID string  `json:"activity_id"`
	Title      string  `json:"title"`
	Topic      string  `json:"topic"`
	Status     string  `json:"status"`
	Accuracy   float64 `json:"accuracy"`
	Attempts   int     `json:"attempts"`
}

// GradebookStudent is one learner's honest gradebook page.
type GradebookStudent struct {
	StudentID string         `json:"student_id"`
	Name      string         `json:"name"`
	Rows      []GradebookRow `json:"rows"`
}

type GradebookService interface {
	ClassGradebook(ctx context.Context, teacherID, classID string) ([]GradebookStudent, error)
	GetNote(ctx context.Context, teacherID, studentID string) (*domain.LearnerNote, error)
	SaveNote(ctx context.Context, teacherID, studentID, note string) (*domain.LearnerNote, error)
}

type gradebookService struct {
	schoolRepo   domain.SchoolRepository
	activityRepo domain.ActivityRepository
	progressRepo domain.ProgressRepository
	noteRepo     domain.NoteRepository
}

func NewGradebookService(
	schoolRepo domain.SchoolRepository,
	activityRepo domain.ActivityRepository,
	progressRepo domain.ProgressRepository,
	noteRepo domain.NoteRepository,
) GradebookService {
	return &gradebookService{
		schoolRepo:   schoolRepo,
		activityRepo: activityRepo,
		progressRepo: progressRepo,
		noteRepo:     noteRepo,
	}
}

// ClassGradebook builds the gradebook for one of the teacher's own classes.
// All numbers are real LearnerActivity rows; a learner with no rows gets an
// empty Rows slice (the UI shows "Not yet assessed").
func (s *gradebookService) ClassGradebook(ctx context.Context, teacherID, classID string) ([]GradebookStudent, error) {
	class, err := s.schoolRepo.FindClassByID(ctx, classID)
	if err != nil || class.TeacherID != teacherID {
		return nil, ErrClassNotFoundForGradebook
	}

	members, err := s.schoolRepo.ClassMembers(ctx, classID)
	if err != nil {
		return nil, err
	}

	activities, err := s.activityRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	learnerIDs := make([]string, 0, len(members))
	for _, m := range members {
		if m.Role == domain.RoleStudent {
			learnerIDs = append(learnerIDs, m.ID)
		}
	}
	learnerActs, err := s.progressRepo.FindLearnerActivitiesBatch(ctx, learnerIDs)
	if err != nil {
		return nil, err
	}

	out := make([]GradebookStudent, 0, len(learnerIDs))
	for _, m := range members {
		if m.Role != domain.RoleStudent {
			continue
		}
		statusMap := make(map[string]domain.LearnerActivity)
		for _, la := range learnerActs[m.ID] {
			statusMap[la.ActivityID] = la
		}
		rows := make([]GradebookRow, 0, len(activities))
		for _, a := range activities {
			la, ok := statusMap[a.ID]
			row := GradebookRow{
				ActivityID: a.ID,
				Title:      a.Title,
				Topic:      a.Topic,
			}
			if ok {
				row.Status = domain.ResolveActivityStatus(la)
				row.Accuracy = la.Accuracy
				row.Attempts = la.Attempts
			} else {
				row.Status = domain.StatusNotStarted
			}
			rows = append(rows, row)
		}
		out = append(out, GradebookStudent{StudentID: m.ID, Name: m.Name, Rows: rows})
	}
	return out, nil
}

func (s *gradebookService) GetNote(ctx context.Context, teacherID, studentID string) (*domain.LearnerNote, error) {
	scoped, err := s.schoolRepo.StudentInTeacherClasses(ctx, teacherID, studentID)
	if err != nil {
		return nil, err
	}
	if !scoped {
		return nil, ErrNotClassTeacher
	}
	note, err := s.noteRepo.FindNote(ctx, studentID)
	if err != nil {
		return nil, nil // honest null — no note yet
	}
	return note, nil
}

func (s *gradebookService) SaveNote(ctx context.Context, teacherID, studentID, note string) (*domain.LearnerNote, error) {
	scoped, err := s.schoolRepo.StudentInTeacherClasses(ctx, teacherID, studentID)
	if err != nil {
		return nil, err
	}
	if !scoped {
		return nil, ErrNotClassTeacher
	}
	existing, err := s.noteRepo.FindNote(ctx, studentID)
	now := time.Now()
	if err != nil || existing == nil {
		existing = &domain.LearnerNote{
			ID:        GenerateSecureID("note"),
			StudentID: studentID,
			CreatedAt: now,
		}
	}
	existing.TeacherID = teacherID
	existing.Note = note
	existing.UpdatedAt = now
	if err := s.noteRepo.UpsertNote(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}
