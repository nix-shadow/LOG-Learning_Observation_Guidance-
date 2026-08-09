package api

import (
	"fmt"
	"log-backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("super-secret-key-for-mvp-only-use-env-in-prod")

// Simulated in-memory DB for users and OTPs
var users = map[string]*models.User{
	"admin@example.com": {ID: "admin-1", Name: "Admin User", Email: "admin@example.com", Role: models.RoleAdmin},
	"user@example.com":  {ID: "user-123", Name: "Aisha", Email: "user@example.com", Phone: "+9779800000000", Role: models.RoleLearner},
}
var otps = map[string]models.OTPRecord{}

func generateOTP() string {
	// Simple mock OTP for demo
	return "123456"
}

func RequestOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number required"})
		return
	}

	otp := generateOTP()
	otps[req.Phone] = models.OTPRecord{
		Phone:     req.Phone,
		OTP:       otp,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	// In a real app, integrate Twilio/SNS here
	fmt.Printf("[MOCK SMS] Sent OTP %s to %s\n", otp, req.Phone)
	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully"})
}

func VerifyOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		OTP   string `json:"otp" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	record, exists := otps[req.Phone]
	if !exists || record.OTP != req.OTP || time.Now().After(record.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Find or create user
	var user *models.User
	for _, u := range users {
		if u.Phone == req.Phone {
			user = u
			break
		}
	}

	if user == nil {
		user = &models.User{
			ID:         fmt.Sprintf("user-%d", time.Now().Unix()),
			Phone:      req.Phone,
			Role:       models.RoleLearner,
			IsVerified: true,
		}
		users[user.Email] = user // simplified
	}

	token := generateJWT(user)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func GoogleLoginMock(c *gin.Context) {
	// Mocking Google OAuth callback
	var req struct {
		Email string `json:"email" binding:"required"`
		Name  string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, exists := users[req.Email]
	if !exists {
		user = &models.User{
			ID:         fmt.Sprintf("user-%d", time.Now().Unix()),
			Email:      req.Email,
			Name:       req.Name,
			Role:       models.RoleLearner,
			IsVerified: true,
		}
		users[req.Email] = user
	}

	token := generateJWT(user)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email required"})
		return
	}
	// Simulate sending reset link
	fmt.Printf("[MOCK EMAIL] Sent password reset to %s\n", req.Email)
	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent."})
}

func generateJWT(user *models.User) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})
	t, _ := token.SignedString(jwtSecret)
	return t
}

func AuthMiddleware(requiredRole models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Simplified for mock: "Bearer <token>" -> parse token
		tokenString := authHeader[len("Bearer "):]
		token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			role := models.Role(claims["role"].(string))
			if requiredRole != "" && role != requiredRole {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
			c.Set("userID", claims["sub"])
			c.Set("userRole", role)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
		}
	}
}
