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

// sanitizeCSVCell neuters spreadsheet formula injection: a value starting with
// = + - @ tab or CR is prefixed with a single quote so Excel/Sheets treats it
// as text. Phones naturally start with "+" (e.g. +977...), so every field is
// guarded, not just attacker-controlled names.
func sanitizeCSVCell(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	}
	return v
}

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
	skippedClasses := 0
	for _, cls := range classes {
		members, err := h.schoolService.ClassRoster(c.Request.Context(), cls.ID)
		if err != nil {
			// Never drop a whole class silently — count it and report honestly
			// in the audit trail instead of pretending the export is complete.
			skippedClasses++
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

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="students_export.csv"`)
	// UTF-8 BOM so Excel opens the file as UTF-8 (names/classes may be Nepali)
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(c.Writer)
	defer w.Flush()
	_ = w.Write([]string{"name", "email", "phone", "class"})
	for _, r := range rows {
		// Sanitize every field against formula injection.
		if err := w.Write([]string{sanitizeCSVCell(r.Name), sanitizeCSVCell(r.Email), sanitizeCSVCell(r.Phone), sanitizeCSVCell(r.Class)}); err != nil {
			RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to write CSV")
			return
		}
	}
	detail := strconv.Itoa(len(rows)) + " rows"
	if skippedClasses > 0 {
		detail += ", " + strconv.Itoa(skippedClasses) + " classes skipped (failed to load)"
	}
	h.audit(c, "export.students_csv", detail)
}
