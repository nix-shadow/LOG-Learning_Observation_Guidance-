package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"log-backend/internal/domain"
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

// callerID resolves the authenticated learner from the request context.
// AuthMiddleware always sets it; returning false here means the route was
// reached without a valid identity, so the caller gets a 401 instead of
// silently falling back to demo data.
func callerID(c *gin.Context) (string, bool) {
	uid, exists := c.Get("userID")
	if !exists {
		return "", false
	}
	id, ok := uid.(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

func (h *LearnerHandler) GetDashboard(c *gin.Context) {
	learnerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}

	user, progress, activities, observations, guidance, err := h.learnerService.GetDashboardData(c.Request.Context(), learnerID)
	if err != nil {
		RespondError(c, http.StatusNotFound, "Not Found", "Learner account not found")
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
	learnerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}

	activities, err := h.learnerService.GetLearningJourneyData(c.Request.Context(), learnerID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load learning journey")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
	})
}

func (h *LearnerHandler) GetChartData(c *gin.Context) {
	learnerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}

	chartData, err := h.learnerService.GetChartData(c.Request.Context(), learnerID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load chart data")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activity_data": chartData,
	})
}

// CompleteActivityRequest carries the real attempt facts a learner produces.
// All fields optional: legacy/offline clients may complete without quiz data.
type CompleteActivityRequest struct {
	ElapsedSeconds int `json:"elapsed_seconds"`
	CorrectCount   int `json:"correct_count"`
	TotalCount     int `json:"total_count"`
}

func (h *LearnerHandler) CompleteActivity(c *gin.Context) {
	activityID := c.Param("id")
	learnerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}

	// Absent body (legacy client) is fine — stats default to zero.
	var req CompleteActivityRequest
	if c.Request.Body != nil {
		raw, _ := io.ReadAll(c.Request.Body)
		if len(bytes.TrimSpace(raw)) > 0 {
			if err := json.Unmarshal(raw, &req); err != nil {
				RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid attempt payload")
				return
			}
		}
	}

	stats := domain.AttemptStats{
		ElapsedSeconds: req.ElapsedSeconds,
		CorrectCount:   req.CorrectCount,
		TotalCount:     req.TotalCount,
	}

	obs, gui, err := h.learnerService.CompleteActivity(c.Request.Context(), learnerID, activityID, stats)
	if err != nil {
		RespondError(c, http.StatusNotFound, "Not Found", "Activity not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Activity completed successfully",
		"observation": obs,
		"guidance":    gui,
		"attempt": gin.H{
			"accuracy":        stats.Accuracy(),
			"score":           stats.Score(),
			"elapsed_seconds": stats.ElapsedSeconds,
		},
	})
}

func (h *LearnerHandler) GetCourses(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	courses, total, err := h.courseService.GetCourses(c.Request.Context(), page, limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to fetch courses")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"courses": courses,
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
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to fetch micro modules")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"modules": modules,
	})
}

func (h *LearnerHandler) GetModeratorRoster(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	roster, total, needsAttention, assignmentsDue, err := h.moderatorService.GetModeratorRoster(c.Request.Context(), page, limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to fetch roster")
		return
	}

	// Derive the class label from real catalog data (first course) instead of
	// a hardcoded string — no fabricated values.
	className := "My Class"
	if courses, _, err := h.courseService.GetCourses(c.Request.Context(), 1, 1); err == nil && len(courses) > 0 {
		className = courses[0].Title
	}

	c.JSON(http.StatusOK, gin.H{
		"class_name":      className,
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
