package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// CSV export (honest, real data only)
// ---------------------------------------------------------------------------

func (h *SchoolHandler) ExportStudentsCSV(c *gin.Context) {
	classes, err := h.schoolService.ListClasses(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to export data")
		return
	}

	// Build a map of user ID -> class names (enrollment is real data).
	type exportRow struct {
		Name  string
		Email string
		Phone string
		Class string
	}
	var rows []exportRow
	seen := make(map[string]bool)
	for _, cls := range classes {
		members, err := h.schoolService.ClassRoster(c.Request.Context(), cls.ID)
		if err != nil {
			continue
		}
		for _, m := range members {
			phone := ""
			if m.Phone != nil {
				phone = *m.Phone
			}
			if !seen[m.ID] {
				seen[m.ID] = true
				rows = append(rows, exportRow{Name: m.Name, Email: m.Email, Phone: phone, Class: cls.Name})
			} else {
				// Learner in multiple classes: append class to existing row
				for i := range rows {
					if rows[i].Email == m.Email {
						rows[i].Class += "; " + cls.Name
						break
					}
				}
			}
		}
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="students_export.csv"`)
	w := csv.NewWriter(c.Writer)
	defer w.Flush()
	_ = w.Write([]string{"name", "email", "phone", "class"})
	for _, r := range rows {
		if err := w.Write([]string{r.Name, r.Email, r.Phone, r.Class}); err != nil {
			RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to write CSV")
			return
		}
	}
	h.audit(c, "export.students_csv", strconv.Itoa(len(rows))+" rows")
}
