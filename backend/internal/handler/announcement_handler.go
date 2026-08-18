package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Announcements (any authenticated role can read; admin + moderator write)
// ---------------------------------------------------------------------------

type createAnnouncementRequest struct {
	Title string `json:"title" binding:"required,min=3,max=200"`
	Body  string `json:"body" binding:"required"`
}

func (h *SchoolHandler) CreateAnnouncement(c *gin.Context) {
	var req createAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "title and body are required")
		return
	}
	userID, _ := c.Get("userID")
	ann, err := h.schoolService.CreateAnnouncement(c.Request.Context(), req.Title, req.Body, userID.(string))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to create announcement")
		return
	}
	h.audit(c, "announcement.create", ann.ID+" "+ann.Title)
	c.JSON(http.StatusCreated, ann)
}

func (h *SchoolHandler) ListAnnouncements(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}
	anns, err := h.schoolService.ListAnnouncements(c.Request.Context(), limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load announcements")
		return
	}
	c.JSON(http.StatusOK, gin.H{"announcements": anns})
}
