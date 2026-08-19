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
	// WP-0.2 C1: offset pagination — never an unbounded row load. The UI can
	// page with limit+offset; total is returned for page-count math.
	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}
	entries, total, err := h.schoolService.ListAuditLogs(c.Request.Context(), limit, offset)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load audit log")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"audit_logs": entries,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
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
	// Same Clear-Site-Data directive as single-token logout (WP-0.1
	// enforcement round): every session is revoked, so the browser should
	// drop the origin's cached responses on every device.
	c.Writer.Header().Add("Clear-Site-Data", `"cache", "cookies", "storage"`)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out on all devices"})
}
