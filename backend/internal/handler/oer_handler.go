package handler

import (
	"net/http"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// OERHandler (WP-3.1 RC-07): batch import of curated OER content packs.
type OERHandler struct {
	oerService    service.OERService
	schoolService service.SchoolService
}

func NewOERHandler(oerService service.OERService, schoolService service.SchoolService) *OERHandler {
	return &OERHandler{oerService: oerService, schoolService: schoolService}
}

// ImportPack accepts one OER pack, validates every row's license against the
// allowlist, imports the valid rows, and returns an honest per-row report.
// Every import is audit-logged with the pack name + counts.
func (h *OERHandler) ImportPack(c *gin.Context) {
	var req domain.OERPack
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid import payload: "+err.Error())
		return
	}
	if req.Name == "" {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Pack name is required")
		return
	}
	if len(req.Activities) == 0 {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Pack has no activities")
		return
	}

	report, err := h.oerService.ImportPack(c.Request.Context(), req)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Import failed")
		return
	}

	userID, _ := c.Get("userID")
	h.schoolService.WriteAuditLog(c.Request.Context(), userID.(string), "oer.import",
		"pack="+req.Name+" imported="+itoa(report.Imported)+" skipped="+itoa(report.Skipped)+" rejected="+itoa(len(report.Errors)), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"pack":     req.Name,
		"imported": report.Imported,
		"skipped":  report.Skipped,
		"rejected": len(report.Errors),
		"errors":   report.Errors,
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}