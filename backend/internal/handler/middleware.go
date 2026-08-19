package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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
	limit    int
	window   time.Duration
}

type visitorEntry struct {
	tokens    int
	lastReset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitorEntry),
		limit:    limit,
		window:   window,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-2 * rl.window)
		for ip, v := range rl.visitors {
			if v.lastReset.Before(cutoff) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// allow consumes one token from the caller's bucket and reports how many
// remain. Remaining is exposed to clients via X-RateLimit-Remaining so the
// offline layer (and school proxies) can back off before hitting the wall.
func (rl *rateLimiter) allow(ip string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists || now.Sub(v.lastReset) > rateLimitWindow {
		rl.visitors[ip] = &visitorEntry{tokens: rateLimitMax - 1, lastReset: now}
		return true, rateLimitMax - 1
	}

	if v.tokens <= 0 {
		return false, 0
	}

	v.tokens--
	return true, v.tokens
}

// Shared bucket for callers that use the default middleware (kept for
// compatibility; main.go now wires per-route budgets).
const (
	rateLimitMax    = 5
	rateLimitWindow = 1 * time.Minute
	cleanupInterval = 5 * time.Minute
)

var authLimiter = newRateLimiter(rateLimitMax, rateLimitWindow)

// Per-route budgets. A single 5/min bucket shared by every auth route meant a
// class of 30 students behind one school IP had ~25 logins rejected, and one
// hammered route (OTP) starved the others. Each route now gets its own bucket
// sized to its purpose: credential checks stay tight, OTP verification (the
// classroom burst case) gets headroom.
const (
	RateLimitLogin      = 10 // email+password / google: slow brute-force bar
	RateLimitRequestOTP = 5  // re-request window is 60s server-side anyway
	RateLimitVerifyOTP  = 20 // a whole class verifying OTPs in one minute
	RateLimitPassword   = 10

	// Privacy endpoints (WP-0.1): consent and erasure are low-frequency by
	// nature; export is read-heavy so its bucket is tighter per IP.
	RateLimitPrivacyWrite  = 10
	RateLimitPrivacyExport = 5
)

// NewLimiter builds a per-route token bucket (main.go wires one per auth route).
func NewLimiter(limit int, window time.Duration) *rateLimiter {
	return newRateLimiter(limit, window)
}

// RateLimitMiddleware keeps the original shared-bucket behavior for callers
// that do not need a per-route budget.
func RateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddlewareWith(authLimiter)
}

// RateLimitMiddlewareWith limits per client IP with the given bucket. WP-0.1
// enhancement (research round): 429s carry Retry-After (RFC 6585), every
// response carries X-RateLimit-Remaining and X-RateLimit-Limit (RFC 9110
// draft-7 headers), so clients can pace themselves honestly.
func RateLimitMiddlewareWith(rl *rateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		allowed, remaining := rl.allow(ip)
		c.Writer.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		c.Writer.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if !allowed {
			// Honest window: the bucket refills one token per minute, so the
			// retry horizon is one window regardless of which token was spent.
			c.Writer.Header().Set("Retry-After", strconv.Itoa(int(rateLimitWindow.Seconds())))
			RespondError(c, http.StatusTooManyRequests, "Too Many Requests", "Too many requests. Please wait before trying again.")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireConsent is the server-side consent gate (WP-0.1 enforcement round).
// The login UI already requires guardian consent to register; this closes the
// bypass path — a raw API client cannot write learner data without an active,
// evidenced guardian grant. Staff roles are exempt (they are not learners
// under the consent regime). The 403 carries a machine-readable code so the
// offline queue preserves records instead of treating the rejection as
// terminal (deleting queued learner work would violate AGENTS.md §3).
func RequireConsent(privacyRepo domain.PrivacyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != domain.RoleStudent {
			c.Next()
			return
		}
		uid, _ := c.Get("userID")
		userID, _ := uid.(string)
		if userID == "" {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authenticated user not found")
			c.Abort()
			return
		}
		granted, err := privacyRepo.HasActiveConsent(c.Request.Context(), userID, domain.ConsentTypeGuardian)
		if err != nil {
			// A failed consent lookup is a server fault. 503 beats silently
			// allowing data collection without evidence.
			RespondError(c, http.StatusServiceUnavailable, "Service Unavailable", "Could not verify consent. Please try again.")
			c.Abort()
			return
		}
		if !granted {
			c.JSON(http.StatusForbidden, gin.H{
				"error":  "Guardian consent is required for this action.",
				"code":   "consent_required",
				"detail": "Please grant guardian consent in Settings, then try again.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func AuthMiddleware(authRepo domain.AuthRepository, userRepo domain.UserRepository, schoolRepo domain.SchoolRepository, requiredRole domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Authorization header missing or invalid")
			c.Abort()
			return
		}

		tokenString := authHeader[7:]

		// Retrieve secret from service package or inject it. Here we use an env fallback.
		// For clean architecture, we should inject this, but for simplicity we can parse it here.
		// To avoid circular dependency, we'll just parse the JWT.
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(service.GetJWTSecret()), nil // Need to export this in service/auth_utils.go
		}, jwt.WithExpirationRequired())

		if err != nil || !token.Valid {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Invalid token claims")
			c.Abort()
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Invalid user ID in token")
			c.Abort()
			return
		}

		roleStr, _ := claims["role"].(string)
		jti, jtiOk := claims["jti"].(string)
		exp, _ := claims["exp"].(float64)

		// The server is the only signer and always sets jti (GenerateJWT) —
		// a token without one cannot be revoked and is rejected outright.
		if !jtiOk || jti == "" {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Invalid token claims")
			c.Abort()
			return
		}
		blocked, err := authRepo.IsTokenBlocked(c.Request.Context(), jti)
		if err != nil || blocked {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Token has been revoked. Please log in again.")
			c.Abort()
			return
		}

		// Revalidate the identity against the database: the token role may be
		// stale (e.g. a user was demoted after the token was issued), and a
		// soft-deleted account must lose access immediately.
		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Account no longer exists")
			c.Abort()
			return
		}
		if user.Role != domain.Role(roleStr) {
			RespondError(c, http.StatusUnauthorized, "Unauthorized", "Role changed. Please log in again.")
			c.Abort()
			return
		}

		role := user.Role

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

		// Global session revocation ("log out on all devices"): tokens issued
		// before the revocation timestamp are rejected even though unexpired.
		if schoolRepo != nil {
			if iat, ok := claims["iat"].(float64); ok {
				if revocation, err := schoolRepo.RevokedBefore(c.Request.Context(), userID); err == nil && revocation.RevokedBefore.After(time.Unix(int64(iat), 0)) {
					RespondError(c, http.StatusUnauthorized, "Unauthorized", "Session revoked. Please log in again.")
					c.Abort()
					return
				}
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
