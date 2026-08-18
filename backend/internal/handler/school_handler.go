package handler

import (
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// SchoolHandler is the shared seam for the school module. Per-resource
// handlers live in their own files so one concept stays in one place:
//
//	class_handler.go       — classes, enrollment, rosters
//	assignment_handler.go  — assignments + submissions
//	announcement_handler.go— announcements
//	audit_handler.go       — audit log + session revocation
//	export_handler.go      — CSV export
type SchoolHandler struct {
	schoolService service.SchoolService
}

func NewSchoolHandler(schoolService service.SchoolService) *SchoolHandler {
	return &SchoolHandler{schoolService: schoolService}
}

func (h *SchoolHandler) actor(c *gin.Context) (string, string) {
	userID, _ := c.Get("userID")
	ip := c.ClientIP()
	return userID.(string), ip
}

func (h *SchoolHandler) audit(c *gin.Context, action, detail string) {
	userID, ip := h.actor(c)
	h.schoolService.WriteAuditLog(c.Request.Context(), userID, action, detail, ip)
}
