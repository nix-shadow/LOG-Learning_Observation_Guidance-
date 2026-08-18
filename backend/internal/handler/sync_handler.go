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
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid sync payload format")
		return
	}

	// Resolve authenticated caller — scoping prevents cross-user data tampering
	callerID, ok := callerID(c)
	if !ok {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
		return
	}

	processedCount, failedCount, err := h.syncService.ProcessBulkSync(c.Request.Context(), callerID, payload.Data)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to sync offline data")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully synced %d offline actions.", processedCount),
		"count":   processedCount,
		"failed":  failedCount,
	})
}
