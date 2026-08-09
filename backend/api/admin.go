package api

import (
	"log-backend/database"
	"log-backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetAdminDashboard(c *gin.Context) {
	var totalUsers int64
	var totalActivities int64

	database.DB.Model(&models.User{}).Count(&totalUsers)
	database.DB.Model(&models.Activity{}).Count(&totalActivities)

	var recentUsers []models.User
	database.DB.Order("created_at desc").Limit(5).Find(&recentUsers)

	c.JSON(http.StatusOK, gin.H{
		"analytics": models.SystemAnalytics{
			TotalUsers:       int(totalUsers),
			ActiveDaily:      int(totalUsers) / 2,
			TotalCompletions: int(totalActivities) * 20,
		},
		"recent_users": recentUsers,
	})
}

func GetUsers(c *gin.Context) {
	var users []models.User
	database.DB.Find(&users)
	c.JSON(http.StatusOK, gin.H{"users": users})
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

	var user models.User
	if err := database.DB.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Role = req.Role
	database.DB.Save(&user)
	c.JSON(http.StatusOK, gin.H{"message": "Role updated", "user": user})
}

func CreateActivity(c *gin.Context) {
	var req models.Activity
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}
	req.CreatedAt = time.Now()
	database.DB.Create(&req)
	c.JSON(http.StatusCreated, req)
}
