package api

import (
	"log-backend/database"
	"log-backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDashboard(c *gin.Context) {
	// For MVP, just fetch the seeded student (user-123). In reality, use c.GetString("userID")
	learnerID := "user-123"

	var user models.User
	var progress models.Progress
	var activities []models.Activity
	var observations []models.Observation
	var guidance []models.Guidance

	database.DB.First(&user, "id = ?", learnerID)
	database.DB.First(&progress, "learner_id = ?", learnerID)
	database.DB.Find(&activities)
	database.DB.Find(&observations, "learner_id = ?", learnerID)
	database.DB.Find(&guidance, "learner_id = ?", learnerID)

	c.JSON(http.StatusOK, gin.H{
		"learner":      user,
		"progress":     progress,
		"activities":   activities,
		"observations": observations,
		"guidance":     guidance,
	})
}

func GetLearningJourney(c *gin.Context) {
	var activities []models.Activity
	database.DB.Order("`order` asc").Find(&activities)
	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

func GetChartData(c *gin.Context) {
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
