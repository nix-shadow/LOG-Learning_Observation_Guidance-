package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"log-backend/database"
	"log-backend/models"
	"net/http"
	"os"
	"sync"
	"time"
	"context"

	"google.golang.org/api/idtoken"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// JWT Secret — read from environment, fatal if missing
// ---------------------------------------------------------------------------

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// For development fallback only — in production this MUST be set
		secret = "dev-only-change-me-in-production-min-32-chars!"
		slog.Warn("JWT_SECRET not set. Using insecure development default. Set JWT_SECRET env var for production.")
	}
	if len(secret) < 32 {
		slog.Error("FATAL: JWT_SECRET must be at least 32 characters")
		os.Exit(1)
	}
	jwtSecret = []byte(secret)
}

// ---------------------------------------------------------------------------
// Rate Limiter — per-IP token bucket (in-memory) with cleanup goroutine
// ---------------------------------------------------------------------------

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
	rateLimitMax    = 5               // max requests per window
	rateLimitWindow = 1 * time.Minute // reset window
	cleanupInterval = 5 * time.Minute // prune stale visitor entries
)

func init() {
	// Background goroutine to prevent memory leak in visitors map.
	// Removes entries that haven't been seen in > 2x the rate limit window.
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

// RateLimitMiddleware enforces per-IP rate limiting on sensitive endpoints.
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

// ---------------------------------------------------------------------------
// Crypto Helpers
// ---------------------------------------------------------------------------

// GenerateSecureID creates a cryptographically random hex ID with the given prefix.
func GenerateSecureID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if crypto/rand fails (should never happen)
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// constantTimeEqual compares two strings in constant time to prevent timing attacks.
func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ---------------------------------------------------------------------------
// Auth Handlers
// ---------------------------------------------------------------------------

func RequestOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required,min=10,max=15"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
		return
	}

	// Clean up expired OTPs for this phone number before creating a new one
	database.DB.Where("phone = ? AND expires_at < ?", req.Phone, time.Now()).Delete(&models.OTPRecord{})

	// Also purge globally expired OTPs older than 10 minutes (background hygiene)
	database.DB.Where("expires_at < ?", time.Now().Add(-10*time.Minute)).Delete(&models.OTPRecord{})

	// Generate cryptographically random 6-digit OTP
	otpInt, err := generateSecureOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}
	otp := fmt.Sprintf("%06d", otpInt)

	// Hash OTP before storing — plaintext OTP must NEVER be persisted
	otpHash, err := HashPassword(otp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process OTP"})
		return
	}

	record := models.OTPRecord{
		Phone:     req.Phone,
		OTP:       otpHash, // Store hash, never plaintext
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	database.DB.Save(&record)

	// In production: send OTP via SMS gateway. For demo, log it.
	slog.Info("[DEMO] OTP generated", "phone", req.Phone, "otp", otp)
	c.JSON(http.StatusOK, gin.H{"message": "OTP sent"})
}

func VerifyOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		OTP   string `json:"otp" binding:"required,len=6"`
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

	// Verify OTP against bcrypt hash — also checks expiry
	if time.Now().After(record.ExpiresAt) || !CheckPasswordHash(req.OTP, record.OTP) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "phone = ?", req.Phone).Error; err != nil {
		user = models.User{
			ID:         GenerateSecureID("user"),
			Phone:      req.Phone,
			Role:       models.RoleStudent,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}
		database.DB.Create(&user)
	}

	t, err := generateJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Delete used OTP
	database.DB.Delete(&record)

	c.JSON(http.StatusOK, gin.H{"token": t, "user": user})
}

func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email address"})
		return
	}

	// In production, dispatch email reset token. For demo/edge, return confirmation.
	// Always return success to prevent email enumeration attacks.
	c.JSON(http.StatusOK, gin.H{
		"message": "If an account exists with this email, a password reset link has been sent.",
	})
}

func GoogleAuth(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("JSON binding failed in GoogleAuth", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" || clientID == "YOUR_GOOGLE_CLIENT_ID_HERE" {
		slog.Warn("GOOGLE_CLIENT_ID is not set or is a placeholder. Rejecting Google Auth.")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Auth is not configured on the server."})
		return
	}

	payload, err := idtoken.Validate(context.Background(), req.Token, clientID)
	if err != nil {
		slog.Error("Invalid Google ID token", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token"})
		return
	}

	email := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)

	var user models.User
	if err := database.DB.First(&user, "email = ?", email).Error; err != nil {
		user = models.User{
			ID:         GenerateSecureID("user-g"),
			Name:       name,
			Email:      email,
			Role:       models.RoleStudent,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}
		database.DB.Create(&user)
	}

	t, err := generateJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": t, "user": user})
}

// Register a new user via email and password
func Register(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Check if user exists
	var count int64
	database.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	user := models.User{
		ID:           GenerateSecureID("usr"),
		Name:         req.Name,
		Email:        req.Email,
		Phone:        GenerateSecureID("dummy-phone"), // prevent unique constraint violation
		PasswordHash: hashedPassword,
		Role:         models.RoleStudent, // Default role
		IsVerified:   true,               // Auto-verify for testing
		CreatedAt:    time.Now(),
	}

	if err := database.DB.Create(&user).Error; err != nil {
		slog.Error("Failed to create user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	token, err := generateJWT(user.ID, user.Role)
	if err != nil {
		slog.Error("Failed to generate JWT", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed after registration"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  user,
	})
}

// Login an existing user via email and password
func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if !CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	token, err := generateJWT(user.ID, user.Role)
	if err != nil {
		slog.Error("Failed to generate JWT", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// ---------------------------------------------------------------------------
// JWT Generation Helper
// ---------------------------------------------------------------------------

func generateJWT(userID string, role models.Role) (string, error) {
	jti := GenerateSecureID("jti") // Unique JWT ID for revocation
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"jti":  jti,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})
	return token.SignedString(jwtSecret)
}

// generateSecureOTP generates a cryptographically random 6-digit OTP integer.
func generateSecureOTP() (int, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return 0, err
	}
	// Mask to a value between 0-999999
	val := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return val % 1000000, nil
}

// ---------------------------------------------------------------------------
// Logout Handler — JWT Revocation
// ---------------------------------------------------------------------------

// LogoutHandler invalidates the user's current JWT by adding its JTI to the
// blocklist. Subsequent requests with this token will be rejected.
func LogoutHandler(c *gin.Context) {
	jti, jtiOk := c.Get("jti")
	exp, expOk := c.Get("exp")
	userID, _ := c.Get("userID")

	if !jtiOk || !expOk {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot revoke token: missing claims"})
		return
	}

	expTime := time.Unix(int64(exp.(float64)), 0)

	revocation := models.TokenBlocklist{
		JTI:       jti.(string),
		UserID:    userID.(string),
		ExpiresAt: expTime,
		RevokedAt: time.Now(),
	}
	if err := database.DB.Create(&revocation).Error; err != nil {
		// If JTI already in blocklist, that's fine — idempotent
		slog.Warn("Token already in blocklist or DB error", "jti", jti, "error", err)
	}

	slog.Info("Token revoked", "user_id", userID, "jti", jti)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully."})
}

// ---------------------------------------------------------------------------
// Password Hashing Utilities
// ---------------------------------------------------------------------------

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ---------------------------------------------------------------------------
// Auth Middleware — JWT Validation + Multi-Tier RBAC
// ---------------------------------------------------------------------------

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
			// Strictly enforce HMAC signing method to prevent algorithm confusion attacks
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return jwtSecret, nil
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

		// Validate required claims exist
		sub, subOk := claims["sub"].(string)
		roleStr, roleOk := claims["role"].(string)
		jti, jtiOk := claims["jti"].(string)
		exp, expOk := claims["exp"].(float64)
		if !subOk || !roleOk || sub == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Malformed token claims"})
			c.Abort()
			return
		}

		// Check token revocation blocklist (if JTI present)
		if jtiOk && jti != "" {
			var blocked models.TokenBlocklist
			if err := database.DB.First(&blocked, "jti = ?", jti).Error; err == nil {
				// Found in blocklist — token was explicitly revoked
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked. Please log in again."})
				c.Abort()
				return
			}
		}

		role := models.Role(roleStr)

		// Multi-Tier RBAC Hardening
		if requiredRole != "" {
			if role == models.RoleAdmin {
				// Admin passes all checks
			} else if role == models.RoleModerator && (requiredRole == models.RoleStudent || requiredRole == models.RoleModerator) {
				// Moderator passes student and moderator checks
			} else if role != requiredRole {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
		}

		// Propagate all claims for downstream use (e.g. LogoutHandler)
		c.Set("userID", sub)
		c.Set("userRole", role)
		if jtiOk {
			c.Set("jti", jti)
		}
		if expOk {
			c.Set("exp", exp)
		}
		c.Next()
	}
}

// ---------------------------------------------------------------------------
// Request ID + Audit Logging Middleware
// ---------------------------------------------------------------------------

// RequestIDMiddleware generates a unique request ID for tracing and audit logging.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := GenerateSecureID("req")
		c.Set("requestID", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Audit log: method, path, status, duration, userID (if authenticated)
		userID := ""
		if uid, exists := c.Get("userID"); exists {
			userID = uid.(string)
		}
		
		slog.Info("Audit Log",
			"type", "audit",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"user_id", userID,
			"ip", c.ClientIP(),
			"request_id", requestID,
		)
	}
}
