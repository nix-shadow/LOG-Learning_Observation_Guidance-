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
