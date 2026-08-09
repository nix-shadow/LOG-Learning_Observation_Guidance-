package api

import (
	"fmt"
	"log-backend/database"
	"log-backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("super-secret-key-harden-later-123456789")

func RequestOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required,min=10,max=15"` // Input validation
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
		return
	}

	otp := "123456" // Mock OTP for Demo. In prod, generate cryptographically secure random numbers.
	record := models.OTPRecord{
		Phone:     req.Phone,
		OTP:       otp,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	database.DB.Save(&record)

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent"})
}

func VerifyOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		OTP   string `json:"otp" binding:"required,len=6"` // Input validation
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var record models.OTPRecord
	if err := database.DB.First(&record, "phone = ?", req.Phone).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
		return
	}

	if record.OTP != req.OTP || time.Now().After(record.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "phone = ?", req.Phone).Error; err != nil {
		user = models.User{
			ID:         fmt.Sprintf("user-%d", time.Now().Unix()),
			Phone:      req.Phone,
			Role:       models.RoleStudent,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}
		database.DB.Create(&user)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})
	t, _ := token.SignedString(jwtSecret)

	database.DB.Delete(&record)

	c.JSON(http.StatusOK, gin.H{"token": t, "user": user})
}

// Password hashing utility for future expansion
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func AuthMiddleware(requiredRole models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or malformed token"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			role := models.Role(claims["role"].(string))

			// Multi-Tier RBAC Hardening
			if requiredRole != "" {
				if role == models.RoleAdmin {
					// Admin passes
				} else if role == models.RoleModerator && (requiredRole == models.RoleStudent || requiredRole == models.RoleModerator) {
					// Moderator passes
				} else if role != requiredRole {
					c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
					c.Abort()
					return
				}
			}

			c.Set("userID", claims["sub"])
			c.Set("userRole", role)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
		}
	}
}
