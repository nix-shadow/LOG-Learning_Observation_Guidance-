package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService   service.AuthService
	schoolService service.SchoolService
}

func NewAuthHandler(authService service.AuthService, schoolService service.SchoolService) *AuthHandler {
	return &AuthHandler{authService: authService, schoolService: schoolService}
}

func (h *AuthHandler) audit(c *gin.Context, action, detail string) {
	userID, _ := c.Get("userID")
	ip := c.ClientIP()
	h.schoolService.WriteAuditLog(c.Request.Context(), userID.(string), action, detail, ip)
}

func (h *AuthHandler) RequestOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required,min=10,max=15"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid phone number format")
		return
	}

	if err := h.authService.RequestOTP(c.Request.Context(), req.Phone); err != nil {
		// The cooldown window is a client-correctable state, not a server
		// failure — a 500 made the UI misleading and the check twice-requestable.
		if errors.Is(err, service.ErrOTPCooldown) {
			RespondError(c, http.StatusTooManyRequests, "Too Many Requests", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to generate OTP")
		return
	}

	slog.Info("OTP generated for phone number", "phone", req.Phone)
	c.JSON(http.StatusOK, gin.H{"message": "OTP sent"})
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		OTP   string `json:"otp" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid request")
		return
	}

	user, token, err := h.authService.VerifyOTP(c.Request.Context(), req.Phone, req.OTP)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "Invalid or expired OTP")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"phone":       user.Phone,
			"role":        user.Role,
			"is_verified": user.IsVerified,
		},
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Password reset instructions sent (Not Implemented in MVP)"})
}

func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid token payload")
		return
	}

	user, token, err := h.authService.GoogleAuth(c.Request.Context(), req.Token)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"role":        user.Role,
			"is_verified": user.IsVerified,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid request format")
		return
	}

	user, token, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func (h *AuthHandler) LogoutHandler(c *gin.Context) {
	jti, exists := c.Get("jti")
	if !exists {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Missing token ID")
		return
	}

	userID, _ := c.Get("userID")
	exp, _ := c.Get("exp")
	expFloat, ok := exp.(float64)
	if !ok {
		// A server-issued token always carries exp; treat an absent claim as
		// already-expired rather than panicking on the type assertion.
		expFloat = 0
	}

	err := h.authService.Logout(c.Request.Context(), jti.(string), userID.(string), expFloat)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to revoke token")
		return
	}

	// Single-token logout is a sensitive operation — the append-only trail
	// should record it just like logout-all does.
	h.audit(c, "auth.logout", "jti="+jti.(string))
	// Clear-Site-Data (WP-0.1 enforcement round): instruct the browser to drop
	// cached responses for this origin on logout so a shared device cannot
	// serve stale authenticated pages (or cached `/me/export` bundles) from
	// the HTTP cache after the session is revoked.
	c.Writer.Header().Add("Clear-Site-Data", `"cache", "cookies", "storage"`)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid input or password too short")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "Unauthorized", "User ID not found in context")
		return
	}

	err := h.authService.UpdatePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
