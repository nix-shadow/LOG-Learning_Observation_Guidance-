package handler

import (
	"errors"
	"net/http"

	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ParentHandler (WP-2.1 RC-04): the school-verified guardian portal.
// Parents get a read-only digest of linked learners; teachers create the
// one-time invites that ARE the school verification.
type ParentHandler struct {
	parentService service.ParentService
	schoolService service.SchoolService
}

func NewParentHandler(parentService service.ParentService, schoolService service.SchoolService) *ParentHandler {
	return &ParentHandler{parentService: parentService, schoolService: schoolService}
}

func (h *ParentHandler) audit(c *gin.Context, userID, action, detail string) {
	ip := c.ClientIP()
	h.schoolService.WriteAuditLog(c.Request.Context(), userID, action, detail, ip)
}

// ParentSignup: POST /api/v1/auth/parent-signup — one atomic flow: create the
// PARENT account, claim the teacher-issued invite code, record the
// parent_access consent with the notice's disclosure hash.
func (h *ParentHandler) ParentSignup(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Email          string `json:"email" binding:"required,email"`
		Password       string `json:"password" binding:"required,min=8"`
		InviteCode     string `json:"invite_code" binding:"required"`
		DisclosureHash string `json:"disclosure_hash" binding:"required"`
		Language       string `json:"language"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid request format")
		return
	}

	user, token, err := h.parentService.ParentSignup(c.Request.Context(), req.Name, req.Email, req.Password, req.InviteCode, req.DisclosureHash, req.Language)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrParentEmailTaken):
			RespondError(c, http.StatusConflict, "Conflict", err.Error())
		case errors.Is(err, service.ErrParentInviteNotFound):
			RespondError(c, http.StatusNotFound, "Not Found", err.Error())
		case errors.Is(err, service.ErrInvalidDisclosure):
			RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		default:
			RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to create parent account")
		}
		return
	}

	h.audit(c, user.ID, "parent.claim", "student-link via invite")
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// CreateParentInvite: POST /api/v1/moderator/students/:id/parent-invite —
// teacher creates the one-time invite code for a student in their class.
func (h *ParentHandler) CreateParentInvite(c *gin.Context) {
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	studentID := c.Param("id")

	link, err := h.parentService.CreateParentInvite(c.Request.Context(), callerID, studentID)
	if err != nil {
		if errors.Is(err, service.ErrNotClassTeacher) {
			RespondError(c, http.StatusNotFound, "Not Found", "Student not found in your classes")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to create parent invite")
		return
	}

	h.audit(c, callerID, "parent.invite_created", "student="+studentID)
	c.JSON(http.StatusOK, gin.H{"invite_code": link.InviteCode})
}

// ListChildren: GET /api/v1/parents/children — minimal identity (id + name),
// no contacts, no OTPs (WP-2.1 privacy boundary).
func (h *ParentHandler) ListChildren(c *gin.Context) {
	parentID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	children, err := h.parentService.LinkedChildren(c.Request.Context(), parentID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load linked children")
		return
	}
	c.JSON(http.StatusOK, gin.H{"children": children})
}

// ChildDigest: GET /api/v1/parents/children/:id/digest — read-only,
// sanitized progress digest.
func (h *ParentHandler) ChildDigest(c *gin.Context) {
	parentID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	studentID := c.Param("id")

	digest, err := h.parentService.ChildDigest(c.Request.Context(), parentID, studentID)
	if err != nil {
		if errors.Is(err, service.ErrParentScope) {
			RespondError(c, http.StatusNotFound, "Not Found", "Learner not linked to this parent")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load progress digest")
		return
	}
	c.JSON(http.StatusOK, digest)
}

// SetDigestOptIn: POST /api/v1/parents/children/:id/opt-in — the parent's
// own preference for receiving the weekly digest (opt-in only).
func (h *ParentHandler) SetDigestOptIn(c *gin.Context) {
	parentID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	studentID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid request format")
		return
	}

	if err := h.parentService.SetDigestOptIn(c.Request.Context(), parentID, studentID, req.Enabled); err != nil {
		if errors.Is(err, service.ErrParentScope) {
			RespondError(c, http.StatusNotFound, "Not Found", "Learner not linked to this parent")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to update preference")
		return
	}

	h.audit(c, parentID, "parent.opt_in", "student="+studentID+" enabled="+boolStr(req.Enabled))
	c.JSON(http.StatusOK, gin.H{"digest_opt_in": req.Enabled})
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
