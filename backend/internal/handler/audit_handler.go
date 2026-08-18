package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Audit log (admin only)
// ---------------------------------------------------------------------------

func (h *SchoolHandler) ListAuditLog(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 200 {
		limit = 50
	}
	entries, err := h.schoolService.ListAuditLogs(c.Request.Context(), limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load audit log")
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": entries})
}

// ---------------------------------------------------------------------------
// Session revocation (any authenticated role)
// ---------------------------------------------------------------------------

func (h *SchoolHandler) LogoutAll(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.schoolService.RevokeAll(c.Request.Context(), userID.(string)); err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to revoke sessions")
		return
	}
	h.audit(c, "auth.logout_all", userID.(string))
	c.JSON(http.StatusOK, gin.H{"message": "Logged out on all devices"})
}
