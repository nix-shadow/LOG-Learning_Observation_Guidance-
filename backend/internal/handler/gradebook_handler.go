package handler

import (
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"

	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GradebookHandler (WP-2.3 RC-08): the honest gradebook. Every number is a
// real stored LearnerActivity row; learners with no rows show "Not yet
// assessed" on the frontend, and the CSV exports only real data.
type GradebookHandler struct {
	gradebookService service.GradebookService
	schoolService    service.SchoolService
}

func NewGradebookHandler(gradebookService service.GradebookService, schoolService service.SchoolService) *GradebookHandler {
	return &GradebookHandler{gradebookService: gradebookService, schoolService: schoolService}
}

func (h *GradebookHandler) audit(c *gin.Context, userID, action, detail string) {
	ip := c.ClientIP()
	h.schoolService.WriteAuditLog(c.Request.Context(), userID, action, detail, ip)
}

// ClassGradebook: GET /api/v1/moderator/gradebook?class_id=...
func (h *GradebookHandler) ClassGradebook(c *gin.Context) {
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	classID := c.Query("class_id")
	if classID == "" {
		RespondError(c, http.StatusBadRequest, "Bad Request", "class_id is required")
		return
	}

	students, err := h.gradebookService.ClassGradebook(c.Request.Context(), callerID, classID)
	if err != nil {
		if errors.Is(err, service.ErrClassNotFoundForGradebook) {
			RespondError(c, http.StatusNotFound, "Not Found", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load gradebook")
		return
	}
	c.JSON(http.StatusOK, gin.H{"students": students})
}

// GradebookCSV: GET /api/v1/moderator/gradebook.csv?class_id=... — real
// data only, every cell sanitized against CSV injection (WP-2.3 RC-08).
func (h *GradebookHandler) GradebookCSV(c *gin.Context) {
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	classID := c.Query("class_id")
	if classID == "" {
		RespondError(c, http.StatusBadRequest, "Bad Request", "class_id is required")
		return
	}

	students, err := h.gradebookService.ClassGradebook(c.Request.Context(), callerID, classID)
	if err != nil {
		if errors.Is(err, service.ErrClassNotFoundForGradebook) {
			RespondError(c, http.StatusNotFound, "Not Found", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load gradebook")
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=gradebook.csv")
	csvW := csv.NewWriter(c.Writer)
	_ = csvW.Write([]string{"student_id", "student_name", "activity_id", "title", "topic", "status", "accuracy", "attempts"})
	for _, s := range students {
		for _, r := range s.Rows {
			_ = csvW.Write([]string{
				sanitizeCSVCell(s.StudentID),
				sanitizeCSVCell(s.Name),
				sanitizeCSVCell(r.ActivityID),
				sanitizeCSVCell(r.Title),
				sanitizeCSVCell(r.Topic),
				sanitizeCSVCell(r.Status),
				strconv.FormatFloat(r.Accuracy, 'f', -1, 64),
				strconv.Itoa(r.Attempts),
			})
		}
	}
	csvW.Flush()

	h.audit(c, callerID, "gradebook.export", "class="+classID)
}

// GetNote: GET /api/v1/moderator/students/:id/note — the teacher's
// annotation on a learner; honest null when none exists yet.
func (h *GradebookHandler) GetNote(c *gin.Context) {
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	studentID := c.Param("id")

	note, err := h.gradebookService.GetNote(c.Request.Context(), callerID, studentID)
	if err != nil {
		if errors.Is(err, service.ErrNotClassTeacher) {
			RespondError(c, http.StatusNotFound, "Not Found", "Student not found in your classes")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load note")
		return
	}
	if note == nil {
		c.JSON(http.StatusOK, gin.H{"note": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"note": note.Note, "updated_at": note.UpdatedAt})
}

// SaveNote: PUT /api/v1/moderator/students/:id/note — one editable,
// supportive annotation per learner (upsert).
func (h *GradebookHandler) SaveNote(c *gin.Context) {
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	studentID := c.Param("id")

	var req struct {
		Note string `json:"note" binding:"required,max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid request format")
		return
	}

	note, err := h.gradebookService.SaveNote(c.Request.Context(), callerID, studentID, req.Note)
	if err != nil {
		if errors.Is(err, service.ErrNotClassTeacher) {
			RespondError(c, http.StatusNotFound, "Not Found", "Student not found in your classes")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to save note")
		return
	}

	h.audit(c, callerID, "note.updated", "student="+studentID)
	c.JSON(http.StatusOK, gin.H{"note": note.Note, "updated_at": note.UpdatedAt})
}
