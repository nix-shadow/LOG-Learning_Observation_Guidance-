package handler

import (
	"errors"
	"net/http"
	"strconv"

	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// PilotHandler (WP-3.3 RC-10): QR poster scans + honest pilot measurement.
// Scans carry no personal data — no IP, no device id, no user id.
type PilotHandler struct {
	pilotService service.PilotService
}

func NewPilotHandler(pilotService service.PilotService) *PilotHandler {
	return &PilotHandler{pilotService: pilotService}
}

// RecordScan is public (a poster QR must work before login) but rate limited
// per IP. The source is one of "qr" | "poster".
func (h *PilotHandler) RecordScan(c *gin.Context) {
	var req struct {
		PosterID string `json:"poster_id" binding:"required"`
		Source   string `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "poster_id is required")
		return
	}
	if req.Source != "" && req.Source != "qr" && req.Source != "poster" {
		RespondError(c, http.StatusBadRequest, "Bad Request", "source must be \"qr\" or \"poster\"")
		return
	}

	scan, err := h.pilotService.RecordScan(c.Request.Context(), req.PosterID, req.Source)
	if err != nil {
		if errors.Is(err, service.ErrPilotPosterNotFound) {
			RespondError(c, http.StatusNotFound, "Not Found", "Poster activity not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to record scan")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"scan_id": scan.ID})
}

// MarkStarted flips a recorded scan to started when the learner clicks
// through — the pilot's honest first-session drop-off signal.
func (h *PilotHandler) MarkStarted(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid scan id")
		return
	}
	if err := h.pilotService.MarkStarted(c.Request.Context(), uint(id)); err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to mark start")
		return
	}
	c.JSON(http.StatusOK, gin.H{"started": true})
}

// Stats is admin-only and returns real aggregates over stored scan rows —
// honest zeros when the pilot has no data yet.
func (h *PilotHandler) Stats(c *gin.Context) {
	stats, err := h.pilotService.Stats(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load pilot stats")
		return
	}
	c.JSON(http.StatusOK, gin.H{"stats": stats})
}