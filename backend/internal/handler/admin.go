package handler

import (
	"errors"
	"net/http"
	"strconv"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminHandler (C3 seam, architecture review): all admin console endpoints
// live behind a struct with an injected AdminService — no handler reaches
// into repositories or the database directly. Mirrors the SchoolHandler
// pattern (struct + constructor) so the console is unit-testable.
type AdminHandler struct {
	adminService service.AdminService
}

func NewAdminHandler(adminService service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	analytics, recentUsers, err := h.adminService.Dashboard(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load analytics")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"analytics":    analytics,
		"recent_users": recentUsers,
	})
}

func (h *AdminHandler) GetUsers(c *gin.Context) {
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

	users, total, consentByUser, err := h.adminService.ListUsers(c.Request.Context(), page, limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load users")
		return
	}

	// Consent status (WP-0.1): principals need to see which learners have
	// guardian consent before relying on their participation. Additive field —
	// null when absent, never a fabricated value.
	type userWithConsent struct {
		domain.User
		Consent *domain.ConsentRecord `json:"consent"`
	}
	response := make([]userWithConsent, 0, len(users))
	for _, u := range users {
		var consent *domain.ConsentRecord
		if rec, ok := consentByUser[u.ID]; ok {
			recCopy := rec
			consent = &recCopy
		}
		response = append(response, userWithConsent{User: u, Consent: consent})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": response,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// UpdateUserRole changes a user's role. The check-then-act last-admin guard,
// the role write, and the audit entry run inside ONE transaction (see
// admin_repo.ChangeRoleTx) so two concurrent demotions can never leave a
// school with zero admins, and a failed write never reports success.
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Role domain.Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid role")
		return
	}

	actor, _ := c.Get("userID")
	user, err := h.adminService.ChangeUserRole(c.Request.Context(), actor.(string), id, c.ClientIP(), req.Role)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"message": "Role updated", "user": user})
	case errors.Is(err, service.ErrLastAdmin):
		RespondError(c, http.StatusBadRequest, "Bad Request", "Cannot demote the last admin. Promote another user to ADMIN first.")
	case errors.Is(err, service.ErrUserNotFound):
		RespondError(c, http.StatusNotFound, "Not Found", "User not found")
	case errors.Is(err, service.ErrInvalidRole):
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid role value. Must be STUDENT, MODERATOR, or ADMIN.")
	default:
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to update role")
	}
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

func (h *AdminHandler) CreateActivity(c *gin.Context) {
	var req CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	actor, _ := c.Get("userID")
	act, err := h.adminService.CreateActivity(c.Request.Context(), actor.(string), c.ClientIP(), service.CreateActivityInput{
		Title:         req.Title,
		Description:   req.Description,
		Topic:         req.Topic,
		Difficulty:    req.Difficulty,
		Prerequisites: req.Prerequisites,
		ContentJSON:   req.ContentJSON,
	})
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to create activity")
		return
	}
	c.JSON(http.StatusCreated, act)
}

// AnalyticsSummary (WP-4.3) serves the opt-in-gated aggregate view: counts
// and averages computed only over learners with an active analytics consent.
// Never learner-level rows — the response cannot leak an individual.
func (h *AdminHandler) AnalyticsSummary(c *gin.Context) {
	sum, err := h.adminService.AnalyticsSummary(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to compute analytics summary")
		return
	}
	c.JSON(http.StatusOK, sum)
}
