package handler

import (
	"net/http"
	"strconv"

	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type LearnerHandler struct {
	learnerService   service.LearnerService
	courseService    service.CourseService
	moderatorService service.ModeratorService
}

func NewLearnerHandler(l service.LearnerService, c service.CourseService, m service.ModeratorService) *LearnerHandler {
	return &LearnerHandler{learnerService: l, courseService: c, moderatorService: m}
}

func (h *LearnerHandler) GetDashboard(c *gin.Context) {
	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	user, progress, activities, observations, guidance, err := h.learnerService.GetDashboardData(c.Request.Context(), learnerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Learner account not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"learner":      user,
		"progress":     progress,
		"activities":   activities,
		"observations": observations,
		"guidance":     guidance,
	})
}

func (h *LearnerHandler) GetLearningJourney(c *gin.Context) {
	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	activities, err := h.learnerService.GetLearningJourneyData(c.Request.Context(), learnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load learning journey"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
	})
}

func (h *LearnerHandler) GetChartData(c *gin.Context) {
	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	chartData, err := h.learnerService.GetChartData(c.Request.Context(), learnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load chart data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activity_data": chartData,
	})
}

func (h *LearnerHandler) CompleteActivity(c *gin.Context) {
	activityID := c.Param("id")
	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	obs, gui, err := h.learnerService.CompleteActivity(c.Request.Context(), learnerID, activityID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Activity completed successfully",
		"observation": obs,
		"guidance":    gui,
	})
}

func (h *LearnerHandler) GetCourses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	courses, total, err := h.courseService.GetCourses(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch courses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": courses,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func (h *LearnerHandler) GetMicroModules(c *gin.Context) {
	activityID := c.Param("id")

	modules, err := h.courseService.GetMicroModules(c.Request.Context(), activityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch micro modules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"modules": modules,
	})
}

func (h *LearnerHandler) GetModeratorRoster(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	roster, total, needsAttention, assignmentsDue, err := h.moderatorService.GetModeratorRoster(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch roster"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"class_name":      "Logic 101: Discrete Structures",
		"active_students": total,
		"needs_attention": needsAttention,
		"assignments_due": assignmentsDue,
		"roster":          roster,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}
