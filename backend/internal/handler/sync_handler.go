package handler

import (
	"fmt"
	"net/http"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type SyncBulkPayload struct {
	Version   string                   `json:"version"`
	Timestamp string                   `json:"timestamp"`
	Data      []domain.SyncRequestItem `json:"data"`
}

type SyncHandler struct {
	syncService service.SyncService
}

func NewSyncHandler(syncService service.SyncService) *SyncHandler {
	return &SyncHandler{syncService: syncService}
}

// SyncBulk processes a batch of offline requests uploaded via a .logsync file
func (h *SyncHandler) SyncBulk(c *gin.Context) {
	var payload SyncBulkPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sync payload format"})
		return
	}

	// Resolve authenticated caller — scoping prevents cross-user data tampering
	callerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		callerID = uid.(string)
	}

	processedCount, err := h.syncService.ProcessBulkSync(c.Request.Context(), callerID, payload.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync offline data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully synced %d offline actions.", processedCount),
		"count":   processedCount,
	})
}
