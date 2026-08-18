package handler

import (
	"github.com/gin-gonic/gin"
)

// ErrorResponse represents an RFC 9457 Problem Details for HTTP APIs
type ErrorResponse struct {
	Type     string `json:"type,omitempty"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// RespondError sends a standardized RFC 9457 error response
func RespondError(c *gin.Context, status int, title, detail string) {
	c.JSON(status, ErrorResponse{
		Type:   "about:blank", // Or a specific URL to your error docs
		Title:  title,
		Status: status,
		Detail: detail,
	})
}
