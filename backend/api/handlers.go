package api

import (
	"log-backend/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"learner":      repository.MockLearner,
		"progress":     repository.MockProgress,
		"activities":   repository.MockActivities,
		"observations": repository.MockObservations,
		"guidance":     repository.MockGuidance,
	})
}

func GetLearningJourney(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"activities": repository.MockActivities,
	})
}

func GetChartData(c *gin.Context) {
	// Provide more realistic data for the recharts implementation
	c.JSON(http.StatusOK, gin.H{
		"activity_data": []map[string]interface{}{
			{"name": "Mon", "score": 65, "duration": 20},
			{"name": "Tue", "score": 70, "duration": 25},
			{"name": "Wed", "score": 68, "duration": 15},
			{"name": "Thu", "score": 75, "duration": 30},
			{"name": "Fri", "score": 85, "duration": 45},
			{"name": "Sat", "score": 82, "duration": 40},
			{"name": "Sun", "score": 88, "duration": 50},
		},
	})
}
