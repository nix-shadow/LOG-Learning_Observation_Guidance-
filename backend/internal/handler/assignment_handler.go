package handler

import (
	"errors"
	"net/http"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Assignments: teacher (moderator) writes, learners submit
// ---------------------------------------------------------------------------

func (h *SchoolHandler) CreateAssignment(c *gin.Context) {
	classID := c.Param("id")
	var req struct {
		Title       string `json:"title" binding:"required,min=3,max=200"`
		Description string `json:"description"`
		ActivityID  string `json:"activity_id"`
		DueDate     string `json:"due_date"` // RFC 3339
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "title is required")
		return
	}
	userID, _ := c.Get("userID")
	assignment, err := h.schoolService.CreateAssignment(c.Request.Context(), classID, req.Title, req.Description, req.ActivityID, userID.(string), req.DueDate)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClassNotFound):
			RespondError(c, http.StatusNotFound, "Not Found", "Class not found")
		case errors.Is(err, service.ErrNotClassTeacher):
			RespondError(c, http.StatusForbidden, "Forbidden", err.Error())
		case errors.Is(err, service.ErrInvalidDueDate):
			RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		default:
			RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to create assignment")
		}
		return
	}
	h.audit(c, "assignment.create", assignment.ID+" class="+classID)
	c.JSON(http.StatusCreated, assignment)
}

func (h *SchoolHandler) ListAssignmentsForClass(c *gin.Context) {
	classID := c.Param("id")
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")
	assignments, err := h.schoolService.AssignmentsForClass(c.Request.Context(), classID, userID.(string), role == domain.RoleAdmin)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClassNotFound):
			RespondError(c, http.StatusNotFound, "Not Found", "Class not found")
		case errors.Is(err, service.ErrNotClassTeacher):
			RespondError(c, http.StatusForbidden, "Forbidden", err.Error())
		default:
			RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load assignments")
		}
		return
	}

	// Attach real submission counts (one batched GROUP BY query, not a COUNT
	// per assignment) so the teacher list never shows a fabricated "0".
	ids := make([]string, 0, len(assignments))
	for _, a := range assignments {
		ids = append(ids, a.ID)
	}
	counts, err := h.schoolService.SubmissionCounts(c.Request.Context(), ids)
	if err != nil {
		counts = map[string]int64{}
	}
	result := make([]gin.H, 0, len(assignments))
	for _, a := range assignments {
		result = append(result, gin.H{
			"id":          a.ID,
			"class_id":    a.ClassID,
			"title":       a.Title,
			"description": a.Description,
			"activity_id": a.ActivityID,
			"due_date":    a.DueDate,
			"submissions": counts[a.ID],
		})
	}
	c.JSON(http.StatusOK, gin.H{"class_id": classID, "assignments": result})
}

// ListMyAssignments returns assignments for all classes the learner belongs to,
// with the learner's own submission status attached.
func (h *SchoolHandler) ListMyAssignments(c *gin.Context) {
	userID, _ := c.Get("userID")
	assignments, err := h.schoolService.AssignmentsForLearner(c.Request.Context(), userID.(string))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load assignments")
		return
	}
	result := make([]gin.H, 0, len(assignments))
	for _, a := range assignments {
		submitted, err := h.schoolService.SubmissionCount(c.Request.Context(), a.ID)
		if err != nil {
			submitted = 0
		}
		result = append(result, gin.H{
			"id":          a.ID,
			"class_id":    a.ClassID,
			"title":       a.Title,
			"description": a.Description,
			"activity_id": a.ActivityID,
			"due_date":    a.DueDate,
			"submissions": submitted,
		})
	}
	c.JSON(http.StatusOK, gin.H{"assignments": result})
}

type submitRequest struct {
	Note string `json:"note" binding:"required,max=4000"`
}

func (h *SchoolHandler) SubmitAssignment(c *gin.Context) {
	assignmentID := c.Param("assignment_id")
	var req submitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "note is required")
		return
	}
	userID, _ := c.Get("userID")
	sub, err := h.schoolService.SubmitAssignment(c.Request.Context(), assignmentID, userID.(string), req.Note)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssignmentNotFound):
			RespondError(c, http.StatusNotFound, "Not Found", "Assignment not found")
		case errors.Is(err, service.ErrNotClassMember):
			RespondError(c, http.StatusForbidden, "Forbidden", err.Error())
		default:
			RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to submit assignment")
		}
		return
	}
	// The upsert may have hit an existing row (offline replay) — return the
	// persisted record so the client keys on the real submission.
	persisted, err := h.schoolService.FindSubmission(c.Request.Context(), sub.AssignmentID, sub.LearnerID)
	if err != nil {
		persisted = sub
	}
	c.JSON(http.StatusOK, gin.H{"message": "Assignment submitted", "submission": persisted})
}

// SubmissionsForAssignment is the teacher view: per-learner submissions for one
// assignment, including the learner name for a readable roster.
func (h *SchoolHandler) SubmissionsForAssignment(c *gin.Context) {
	assignmentID := c.Param("assignment_id")
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")
	subs, err := h.schoolService.SubmissionsForAssignment(c.Request.Context(), assignmentID, userID.(string), role == domain.RoleAdmin)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssignmentNotFound), errors.Is(err, service.ErrClassNotFound):
			RespondError(c, http.StatusNotFound, "Not Found", "Assignment not found")
		case errors.Is(err, service.ErrNotClassTeacher):
			RespondError(c, http.StatusForbidden, "Forbidden", err.Error())
		default:
			RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load submissions")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"assignment_id": assignmentID, "submissions": subs})
}
