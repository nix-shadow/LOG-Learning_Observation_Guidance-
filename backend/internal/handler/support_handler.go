package handler

import (
	"errors"
	"net/http"

	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// SupportHandler (WP-2.2 RC-06): the who-to-call funnel. Learners get a
// wizard → guidance → escalation path; escalated issues land in the
// moderator/admin inbox, and every action is audit-logged.
type SupportHandler struct {
	supportService service.SupportService
	schoolService  service.SchoolService
}

func NewSupportHandler(supportService service.SupportService, schoolService service.SchoolService) *SupportHandler {
	return &SupportHandler{supportService: supportService, schoolService: schoolService}
}

func (h *SupportHandler) audit(c *gin.Context, userID, action, detail string) {
	ip := c.ClientIP()
	h.schoolService.WriteAuditLog(c.Request.Context(), userID, action, detail, ip)
}

// CreateIssue: POST /api/v1/support/issue — category + description; the
// frontend wizard decides `escalated` after the guidance step (never
// escalated = self-served, no inbox noise).
func (h *SupportHandler) CreateIssue(c *gin.Context) {
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}

	var req struct {
		Category    string `json:"category" binding:"required"`
		Description string `json:"description" binding:"required,min=10"`
		Escalated   bool   `json:"escalated"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid request format")
		return
	}

	issue, err := h.supportService.CreateIssue(c.Request.Context(), callerID, req.Category, req.Description, req.Escalated)
	if err != nil {
		if errors.Is(err, service.ErrBadCategory) {
			RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to create support issue")
		return
	}

	h.audit(c, callerID, "support.issue_created", "issue="+issue.ID)
	c.JSON(http.StatusCreated, issue)
}

// MyIssues: GET /api/v1/support/my-issues — the user's own history.
func (h *SupportHandler) MyIssues(c *gin.Context) {
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	issues, err := h.supportService.MyIssues(c.Request.Context(), callerID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load issues")
		return
	}
	c.JSON(http.StatusOK, gin.H{"issues": issues})
}

// Inbox: GET /api/v1/support/inbox — open escalated issues only
// (moderator group: MODERATOR + ADMIN via AuthMiddleware).
func (h *SupportHandler) Inbox(c *gin.Context) {
	issues, err := h.supportService.Inbox(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load inbox")
		return
	}
	c.JSON(http.StatusOK, gin.H{"issues": issues})
}

// ResolveIssue: PUT /api/v1/support/issue/:id — moderator/admin closes an
// escalated issue with a resolution note; audit-logged.
func (h *SupportHandler) ResolveIssue(c *gin.Context) {
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}

	var req struct {
		ResolutionNote string `json:"resolution_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid request format")
		return
	}

	issue, err := h.supportService.ResolveIssue(c.Request.Context(), c.Param("id"), callerID, req.ResolutionNote)
	if err != nil {
		if errors.Is(err, service.ErrIssueNotFound) {
			RespondError(c, http.StatusNotFound, "Not Found", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to resolve issue")
		return
	}

	h.audit(c, callerID, "support.issue_resolved", "issue="+issue.ID)
	c.JSON(http.StatusOK, issue)
}
