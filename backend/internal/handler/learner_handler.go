package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

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
		// as_of (WP-0.2 enforcement round): the server-clock timestamp of this
		// payload. The frontend renders it so a cached dashboard never masquerades
		// as live data — the staleness is visible, not hidden.
		"as_of": time.Now(),
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
		"as_of":         time.Now(),
	})
}

// CompleteActivityRequest carries the real attempt facts a learner produces.
// All fields optional: legacy/offline clients may complete without quiz data.
// completed_at_unix_ms + timezone_iana (WP-0.2 research round) let offline
// completions be dated by the learner's clock (clamped server-side), so a
// flush days later still lands on the right calendar day.
type CompleteActivityRequest struct {
	ElapsedSeconds    int    `json:"elapsed_seconds"`
	CorrectCount      int    `json:"correct_count"`
	TotalCount        int    `json:"total_count"`
	CompletedAtUnixMs int64  `json:"completed_at_unix_ms"`
	TimezoneIANA      string `json:"timezone_iana"`
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

	// Clamp client-reported facts so a hostile/buggy payload (negative
	// elapsed, absurd counts) can never poison analytics or grow rows forever.
	// The completion timestamp is clamped in AttemptStats.CompletedAt.
	stats := domain.AttemptStats{
		ElapsedSeconds:    req.ElapsedSeconds,
		CorrectCount:      req.CorrectCount,
		TotalCount:        req.TotalCount,
		CompletedAtUnixMs: req.CompletedAtUnixMs,
		TimezoneIANA:      req.TimezoneIANA,
	}.Clamp()

	obs, gui, err := h.learnerService.CompleteActivity(c.Request.Context(), learnerID, activityID, stats)
	if err != nil {
		// ONLY a missing activity is a 404 — a transient backend failure is a
		// server error, because the offline queue treats 4xx as terminal and
		// would delete the learner's queued work.
		if errors.Is(err, service.ErrActivityNotFound) {
			RespondError(c, http.StatusNotFound, "Not Found", "Activity not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to record completion")
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

	// WP-0.2 C5: courses are annotated with the caller's real enrollment
	// state; the anonymous path (no userID in context) yields counts only.
	userID, _ := c.Get("userID")
	uid, _ := userID.(string)

	courses, total, err := h.courseService.GetCourses(c.Request.Context(), uid, page, limit)
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

// Enroll persists per-learner enrollment (WP-0.2 C5). Idempotent: enrolling
// twice is not an error — the desired state is reached either way.
func (h *LearnerHandler) Enroll(c *gin.Context) {
	userID, _ := c.Get("userID")
	courseID := c.Param("id")
	if err := h.courseService.Enroll(c.Request.Context(), userID.(string), courseID); err != nil {
		switch {
		case errors.Is(err, service.ErrCourseNotFound):
			RespondError(c, http.StatusNotFound, "Not Found", "Course not found")
		default:
			RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to enroll")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Enrolled", "course_id": courseID, "is_enrolled": true})
}

func (h *LearnerHandler) Unenroll(c *gin.Context) {
	userID, _ := c.Get("userID")
	courseID := c.Param("id")
	if err := h.courseService.Unenroll(c.Request.Context(), userID.(string), courseID); err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to unenroll")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Unenrolled", "course_id": courseID, "is_enrolled": false})
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

	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}

	roster, total, needsAttention, assignmentsDue, className, err := h.moderatorService.GetModeratorRoster(c.Request.Context(), callerID, page, limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to fetch roster")
		return
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
