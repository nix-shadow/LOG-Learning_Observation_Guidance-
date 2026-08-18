package handler

import (
	"net/http"
	"strconv"
	"time"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func GetAdminDashboard(c *gin.Context) {
	var totalUsers int64
	var totalActivities int64
	var totalCompletions int64
	var activeDaily int64

	database.DB.Model(&domain.User{}).Count(&totalUsers)
	database.DB.Model(&domain.Activity{}).Count(&totalActivities)
	database.DB.Model(&domain.Progress{}).Select("COALESCE(SUM(completed), 0)").Scan(&totalCompletions)
	// Active daily = learners with progress activity in the last 24 hours
	database.DB.Model(&domain.User{}).Where("updated_at > ?", time.Now().Add(-24*time.Hour)).Count(&activeDaily)

	var recentUsers []domain.User
	database.DB.Order("created_at desc").Limit(5).Find(&recentUsers)

	c.JSON(http.StatusOK, gin.H{
		"analytics": domain.SystemAnalytics{
			TotalUsers:       int(totalUsers),
			ActiveDaily:      int(activeDaily),
			TotalCompletions: int(totalCompletions),
		},
		"recent_users": recentUsers,
	})
}

func GetUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var users []domain.User
	var total int64

	database.DB.Model(&domain.User{}).Count(&total)
	database.DB.Limit(limit).Offset(offset).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func UpdateUserRole(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Role domain.Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	// Validate role is one of the defined constants — reject any arbitrary string
	switch req.Role {
	case domain.RoleStudent, domain.RoleModerator, domain.RoleAdmin:
		// valid — proceed
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role value. Must be STUDENT, MODERATOR, or ADMIN."})
		return
	}

	var user domain.User
	if err := database.DB.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Never demote the last remaining ADMIN — a school with zero principals
	// has no recovery path (nobody can promote a new one).
	if user.Role == domain.RoleAdmin && req.Role != domain.RoleAdmin {
		var adminCount int64
		database.DB.Model(&domain.User{}).Where("role = ?", domain.RoleAdmin).Count(&adminCount)
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot demote the last admin. Promote another user to ADMIN first."})
			return
		}
	}

	user.Role = req.Role
	database.DB.Save(&user)

	// Append-only audit trail for sensitive privilege changes.
	actor, _ := c.Get("userID")
	database.DB.Create(&domain.AuditLog{
		UserID:    actor.(string),
		Action:    "user.role_change",
		Detail:    id + " -> " + string(req.Role),
		IP:        c.ClientIP(),
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Role updated", "user": user})
}

// CreateActivityRequest is a strict DTO that prevents clients from injecting
// server-managed fields like ID, CreatedAt, or Order.
type CreateActivityRequest struct {
	Title         string `json:"title" binding:"required,min=3,max=200"`
	Description   string `json:"description" binding:"required"`
	Topic         string `json:"topic" binding:"required"`
	Difficulty    string `json:"difficulty" binding:"required,oneof=Beginner Intermediate Advanced"`
	Prerequisites string `json:"prerequisites"`
	ContentJSON   string `json:"content_json"`
}

func CreateActivity(c *gin.Context) {
	var req CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Count existing activities to auto-assign display order
	var count int64
	database.DB.Model(&domain.Activity{}).Count(&count)

	act := domain.Activity{
		ID:            service.GenerateSecureID("act"), // Server-generated ID
		Title:         req.Title,
		Description:   req.Description,
		Topic:         req.Topic,
		Difficulty:    req.Difficulty,
		Prerequisites: req.Prerequisites,
		ContentJSON:   req.ContentJSON,
		Order:         int(count) + 1,
		CreatedAt:     time.Now(),
	}

	if err := database.DB.Create(&act).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create activity"})
		return
	}
	actor, _ := c.Get("userID")
	database.DB.Create(&domain.AuditLog{
		UserID:    actor.(string),
		Action:    "activity.create",
		Detail:    act.ID + " " + act.Title,
		IP:        c.ClientIP(),
		CreatedAt: time.Now(),
	})
	c.JSON(http.StatusCreated, act)
}
