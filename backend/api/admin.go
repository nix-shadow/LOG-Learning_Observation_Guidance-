package api

import (
	"log-backend/models"
	"log-backend/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetAdminDashboard(c *gin.Context) {
	// Simple aggregated analytics for Admin
	totalUsers := len(users)
	totalActivities := len(repository.MockActivities)

	c.JSON(http.StatusOK, gin.H{
		"analytics": models.SystemAnalytics{
			TotalUsers:       totalUsers,
			ActiveDaily:      totalUsers / 2, // Mock math
			TotalCompletions: totalActivities * 20,
		},
		"recent_users": getUsersList(),
	})
}

func getUsersList() []models.User {
	var list []models.User
	for _, u := range users {
		list = append(list, *u)
	}
	return list
}

func GetUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"users": getUsersList()})
}

func UpdateUserRole(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Role models.Role `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	for _, u := range users {
		if u.ID == id {
			u.Role = req.Role
			c.JSON(http.StatusOK, gin.H{"message": "Role updated", "user": u})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
}

func CreateActivity(c *gin.Context) {
	var req models.Activity
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity payload"})
		return
	}

	req.CreatedAt = time.Now()
	// Mock append
	repository.MockActivities = append(repository.MockActivities, req)
	c.JSON(http.StatusCreated, req)
}
