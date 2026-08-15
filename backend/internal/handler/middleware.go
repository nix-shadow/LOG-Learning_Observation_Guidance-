package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitorEntry
}

type visitorEntry struct {
	tokens    int
	lastReset time.Time
}

var authLimiter = &rateLimiter{
	visitors: make(map[string]*visitorEntry),
}

const (
	rateLimitMax    = 5
	rateLimitWindow = 1 * time.Minute
	cleanupInterval = 5 * time.Minute
)

func init() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			authLimiter.mu.Lock()
			cutoff := time.Now().Add(-2 * rateLimitWindow)
			for ip, v := range authLimiter.visitors {
				if v.lastReset.Before(cutoff) {
					delete(authLimiter.visitors, ip)
				}
			}
			authLimiter.mu.Unlock()
		}
	}()
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists || now.Sub(v.lastReset) > rateLimitWindow {
		rl.visitors[ip] = &visitorEntry{tokens: rateLimitMax - 1, lastReset: now}
		return true
	}

	if v.tokens <= 0 {
		return false
	}

	v.tokens--
	return true
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !authLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please wait before trying again.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func AuthMiddleware(authRepo domain.AuthRepository, requiredRole domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or invalid"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]

		// Retrieve secret from service package or inject it. Here we use an env fallback.
		// For clean architecture, we should inject this, but for simplicity we can parse it here.
		// To avoid circular dependency, we'll just parse the JWT.
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(service.GetJWTSecret()), nil // Need to export this in service/auth_utils.go
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		roleStr, _ := claims["role"].(string)
		jti, jtiOk := claims["jti"].(string)
		exp, _ := claims["exp"].(float64)

		if jtiOk && jti != "" {
			blocked, err := authRepo.IsTokenBlocked(c.Request.Context(), jti)
			if err != nil || blocked {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked. Please log in again."})
				c.Abort()
				return
			}
		}

		role := domain.Role(roleStr)

		if requiredRole != "" {
			if role == domain.RoleAdmin {
				// pass
			} else if role == domain.RoleModerator && (requiredRole == domain.RoleStudent || requiredRole == domain.RoleModerator) {
				// pass
			} else if role != requiredRole {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
		}

		c.Set("userID", userID)
		c.Set("role", role)
		c.Set("jti", jti)
		c.Set("exp", exp)

		c.Next()
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := service.GenerateSecureID("req")
		c.Set("RequestID", reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)

		start := time.Now()
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		if !strings.Contains(c.Request.URL.Path, "/api/ping") {
			slog.Info("Request Handled",
				"reqID", reqID,
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"status", status,
				"duration", duration,
			)
		}
	}
}
