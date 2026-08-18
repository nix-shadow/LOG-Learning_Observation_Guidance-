package handler

import (
	"log/slog"
	"net/http"

	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
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

	err := h.authService.Logout(c.Request.Context(), jti.(string), userID.(string), exp.(float64))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to revoke token")
		return
	}

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
