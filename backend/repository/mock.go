package repository

import (
	"log-backend/models"
	"time"
)

var (
	MockLearner = models.User{
		ID:        "user-123",
		Name:      "Aisha",
		Email:     "aisha@example.com",
		Phone:     "+9779800000000",
		Role:      models.RoleLearner,
		CreatedAt: time.Now().AddDate(0, -1, 0),
	}

	MockActivities = []models.Activity{
		{ID: "act-1", Title: "Introduction to Logic", Description: "Basic logic concepts.", Status: "Completed", Topic: "Logic", Order: 1},
		{ID: "act-2", Title: "Boolean Algebra", Description: "Understanding AND, OR, NOT.", Status: "Completed", Topic: "Logic", Order: 2},
		{ID: "act-3", Title: "Problem Solving", Description: "Applying logic to real problems.", Status: "In progress", Topic: "Logic", Order: 3},
		{ID: "act-4", Title: "Advanced Algorithms", Description: "Sorting and searching.", Status: "Not started", Topic: "Algorithms", Order: 4},
	}

	MockProgress = models.Progress{
		LearnerID:     "user-123",
		TotalTopics:   10,
		Completed:     2,
		CurrentStreak: 3,
		OverallScore:  85.5,
	}

	MockObservations = []models.Observation{
		{ID: "obs-1", LearnerID: "user-123", Category: "strengths", Text: "You have a strong grasp of Boolean Algebra concepts.", CreatedAt: time.Now()},
		{ID: "obs-2", LearnerID: "user-123", Category: "areas needing improvement", Text: "Problem Solving exercises take slightly longer than average.", CreatedAt: time.Now()},
		{ID: "obs-3", LearnerID: "user-123", Category: "consistency", Text: "You've been studying consistently for 3 days in a row.", CreatedAt: time.Now()},
	}

	MockGuidance = []models.Guidance{
		{ID: "gui-1", LearnerID: "user-123", Type: "next_step", Text: "Continue with the Problem Solving activity.", Action: "/learning/act-3", CreatedAt: time.Now()},
		{ID: "gui-2", LearnerID: "user-123", Type: "practice", Text: "Review Boolean Algebra truth tables before moving to Advanced Algorithms.", Action: "/learning/act-2", CreatedAt: time.Now()},
		{ID: "gui-3", LearnerID: "user-123", Type: "insight", Text: "Your consistency is helping you retain concepts better.", Action: "", CreatedAt: time.Now()},
	}
)
