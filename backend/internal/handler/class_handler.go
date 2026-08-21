package handler

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

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
			"invite_code":  cls.InviteCode,
			"created_at":   cls.CreatedAt,
			"member_count": count,
		})
	}
	c.JSON(http.StatusOK, gin.H{"classes": result})
}

// ---------------------------------------------------------------------------
// Moderator: classes (WP-1.5 class-creation wizard)
// ---------------------------------------------------------------------------

type createModeratorClassRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=120"`
	Grade   string `json:"grade" binding:"required"`
	Section string `json:"section" binding:"required"`
}

func (h *SchoolHandler) CreateModeratorClass(c *gin.Context) {
	userID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	var req createModeratorClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "name, grade and section are required")
		return
	}
	class, err := h.schoolService.CreateClass(c.Request.Context(), req.Name, req.Grade, req.Section, userID)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	h.audit(c, "class.create", class.ID+" "+class.Name)
	c.JSON(http.StatusCreated, class)
}

// ---------------------------------------------------------------------------
// Learner: join a class by invite code (WP-1.5)
// ---------------------------------------------------------------------------

type joinClassRequest struct {
	Code string `json:"code" binding:"required,min=4,max=10"`
}

func (h *SchoolHandler) JoinClass(c *gin.Context) {
	userID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	var req joinClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "a class code is required")
		return
	}
	class, err := h.schoolService.JoinClassByCode(c.Request.Context(), strings.ToUpper(req.Code), userID)
	if errors.Is(err, service.ErrInviteCodeNotFound) {
		RespondError(c, http.StatusNotFound, "Not Found", "No class found for this code")
		return
	}
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to join class")
		return
	}
	h.audit(c, "class.join", class.ID+" "+class.Name)
	c.JSON(http.StatusOK, gin.H{
		"class_id": class.ID,
		"name":     class.Name,
		"grade":    class.Grade,
		"section":  class.Section,
	})
}

// ---------------------------------------------------------------------------
// Moderator: roster CSV import (WP-1.5)
// ---------------------------------------------------------------------------

func (h *SchoolHandler) ImportClassRoster(c *gin.Context) {
	userID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}
	classID := c.Param("id")

	file, err := c.FormFile("file")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "attach a CSV file as field 'file'")
		return
	}
	f, err := file.Open()
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "could not open uploaded file")
		return
	}
	defer func() { _ = f.Close() }()

	rows, parseErr := parseRosterCSV(f)
	if parseErr != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", parseErr.Error())
		return
	}
	if len(rows) == 0 {
		RespondError(c, http.StatusBadRequest, "Bad Request", "the CSV contains no data rows")
		return
	}

	report, err := h.schoolService.ImportRoster(c.Request.Context(), classID, userID, rows)
	if errors.Is(err, service.ErrClassNotFound) {
		RespondError(c, http.StatusNotFound, "Not Found", "class not found")
		return
	}
	if errors.Is(err, service.ErrNotClassTeacher) {
		RespondError(c, http.StatusForbidden, "Forbidden", "you can only import into your own class")
		return
	}
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to import roster")
		return
	}
	h.audit(c, "roster.import", fmt.Sprintf("%s imported=%d errors=%d", classID, report.Imported, len(report.Errors)))
	c.JSON(http.StatusOK, report)
}

// parseRosterCSV reads a roster CSV with a header row and columns
// name,email,phone (password optional). Returns the 1-based data rows in
// file order; a missing header or an unparseable line fails the whole upload
// (honest — a silently renumbered roster is worse than a clear error).
func parseRosterCSV(r io.Reader) ([]service.RosterImportRow, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, errors.New("could not parse CSV — check quoting and commas")
	}
	if len(records) < 2 {
		return nil, errors.New("CSV must have a header row and at least one data row")
	}
	header := make(map[string]int)
	for i, col := range records[0] {
		header[strings.ToLower(strings.TrimSpace(col))] = i
	}
	nameIdx, okN := header["name"]
	emailIdx, okE := header["email"]
	if !okN || !okE {
		return nil, errors.New("CSV header must include 'name' and 'email' columns (phone and password are optional)")
	}
	phoneIdx := header["phone"]
	passwordIdx := header["password"]

	rows := make([]service.RosterImportRow, 0, len(records)-1)
	for _, rec := range records[1:] {
		if len(rec) == 0 || (len(rec) == 1 && strings.TrimSpace(rec[0]) == "") {
			continue
		}
		row := service.RosterImportRow{
			Name:  getCSVCol(rec, nameIdx),
			Email: getCSVCol(rec, emailIdx),
		}
		if phoneIdx >= 0 && phoneIdx < len(rec) {
			row.Phone = strings.TrimSpace(rec[phoneIdx])
		}
		if passwordIdx >= 0 && passwordIdx < len(rec) {
			row.Password = strings.TrimSpace(rec[passwordIdx])
		}
		rows = append(rows, row)
	}
	// Row numbers in the report are 1-based against the FILE (header = row 1).
	for i := range rows {
		rows[i].RowNo = i + 2
	}
	return rows, nil
}

func getCSVCol(rec []string, idx int) string {
	if idx < 0 || idx >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[idx])
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
