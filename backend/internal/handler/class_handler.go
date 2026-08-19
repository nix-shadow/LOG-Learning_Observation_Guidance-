package handler

import (
	"errors"
	"net/http"
	"strconv"

	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Admin: Classes
// ---------------------------------------------------------------------------

type createClassRequest struct {
	Name      string `json:"name" binding:"required,min=2,max=120"`
	Grade     string `json:"grade" binding:"required"`
	Section   string `json:"section" binding:"required"`
	TeacherID string `json:"teacher_id" binding:"required"`
}

func (h *SchoolHandler) CreateClass(c *gin.Context) {
	var req createClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "name, grade, section and teacher_id are required")
		return
	}
	class, err := h.schoolService.CreateClass(c.Request.Context(), req.Name, req.Grade, req.Section, req.TeacherID)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	h.audit(c, "class.create", class.ID+" "+class.Name)
	c.JSON(http.StatusCreated, class)
}

func (h *SchoolHandler) ListClasses(c *gin.Context) {
	classes, err := h.schoolService.ListClasses(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load classes")
		return
	}
	// Attach a member count to every class so the admin roster view is honest.
	result := make([]gin.H, 0, len(classes))
	for _, cls := range classes {
		count, err := h.schoolService.ClassMemberCount(c.Request.Context(), cls.ID)
		if err != nil {
			count = 0
		}
		result = append(result, gin.H{
			"id":           cls.ID,
			"name":         cls.Name,
			"grade":        cls.Grade,
			"section":      cls.Section,
			"teacher_id":   cls.TeacherID,
			"created_at":   cls.CreatedAt,
			"member_count": count,
		})
	}
	c.JSON(http.StatusOK, gin.H{"classes": result})
}

// ListMyClasses is the teacher (moderator) view: only classes where the caller
// is the assigned teacher, with member counts.
func (h *SchoolHandler) ListMyClasses(c *gin.Context) {
	userID, _ := c.Get("userID")
	classes, err := h.schoolService.ClassesByTeacher(c.Request.Context(), userID.(string))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load classes")
		return
	}
	result := make([]gin.H, 0, len(classes))
	for _, cls := range classes {
		count, err := h.schoolService.ClassMemberCount(c.Request.Context(), cls.ID)
		if err != nil {
			count = 0
		}
		result = append(result, gin.H{
			"id":           cls.ID,
			"name":         cls.Name,
			"grade":        cls.Grade,
			"section":      cls.Section,
			"created_at":   cls.CreatedAt,
			"member_count": count,
		})
	}
	c.JSON(http.StatusOK, gin.H{"classes": result})
}

type enrollRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

func (h *SchoolHandler) EnrollStudents(c *gin.Context) {
	classID := c.Param("id")
	var req enrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "user_ids is required")
		return
	}
	count, err := h.schoolService.EnrollStudents(c.Request.Context(), classID, req.UserIDs)
	if err != nil {
		if errors.Is(err, service.ErrClassNotFound) {
			RespondError(c, http.StatusNotFound, "Not Found", "Class not found")
			return
		}
		RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	h.audit(c, "class.enroll", classID+" members="+strconv.Itoa(count))
	c.JSON(http.StatusOK, gin.H{"message": "Students enrolled", "member_count": count})
}

func (h *SchoolHandler) UnenrollStudent(c *gin.Context) {
	classID := c.Param("id")
	userID := c.Param("user_id")
	if err := h.schoolService.UnenrollStudent(c.Request.Context(), classID, userID); err != nil {
		if errors.Is(err, service.ErrNotEnrolled) {
			RespondError(c, http.StatusNotFound, "Not Found", "Student is not enrolled in this class")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to remove student")
		return
	}
	h.audit(c, "class.unenroll", classID+" user="+userID)
	c.JSON(http.StatusOK, gin.H{"message": "Student removed"})
}

func (h *SchoolHandler) ClassRoster(c *gin.Context) {
	classID := c.Param("id")
	users, err := h.schoolService.ClassRoster(c.Request.Context(), classID)
	if err != nil {
		if errors.Is(err, service.ErrClassNotFound) {
			RespondError(c, http.StatusNotFound, "Not Found", "Class not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load roster")
		return
	}
	c.JSON(http.StatusOK, gin.H{"class_id": classID, "students": users})
}
