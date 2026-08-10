package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log-backend/api"
	"log-backend/database"
	"log-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load configuration
	err := godotenv.Load()
	if err != nil {
		slog.Warn("No .env file found or failed to load. Falling back to environment variables.")
	}

	// Initialize Database
	database.InitDB()

	// Use gin.New() + only the Logger and Recovery middlewares we control
	// so we avoid gin.Default()'s panic recovery leaking stack traces to clients.
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/api/ping"},
	}))
	// Custom recovery: return 500 without exposing internal stack traces
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}))

	// Enforce a global request body size limit (4 MB) to prevent DoS
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20) // 4 MB
		c.Next()
	})

	// ---------------------------------------------------------------------------
	// Global Middleware: Security Headers + CORS
	// ---------------------------------------------------------------------------
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:3000"
		slog.Info("CORS_ORIGIN not set, defaulting to http://localhost:3000")
	}

	r.Use(func(c *gin.Context) {
		// CORS — restricted to configured origin
		c.Writer.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// Security Headers — comprehensive hardening
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' https://fonts.gstatic.com; connect-src 'self' "+corsOrigin)
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Global request ID + audit logging
	r.Use(api.RequestIDMiddleware())

	// ---------------------------------------------------------------------------
	// Public Auth Routes — rate limited
	// ---------------------------------------------------------------------------
	authRoutes := r.Group("/api/auth")
	authRoutes.Use(api.RateLimitMiddleware())
	{
		authRoutes.POST("/request-otp", api.RequestOTP)
		authRoutes.POST("/verify-otp", api.VerifyOTP)
		authRoutes.POST("/forgot-password", api.ForgotPassword)
		authRoutes.POST("/google", api.GoogleAuth)
	}

	// ---------------------------------------------------------------------------
	// Public Health Probe
	// ---------------------------------------------------------------------------
	r.GET("/api/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })

	// ---------------------------------------------------------------------------
	// Protected Student Routes — requires valid JWT with STUDENT role minimum
	// ---------------------------------------------------------------------------
	apiRoutes := r.Group("/api")
	apiRoutes.Use(api.AuthMiddleware(models.RoleStudent))
	{
		apiRoutes.GET("/dashboard", api.GetDashboard)
		apiRoutes.GET("/learning-journey", api.GetLearningJourney)
		apiRoutes.GET("/chart-data", api.GetChartData)
		apiRoutes.GET("/courses", api.GetCourses)
		apiRoutes.GET("/activities/:id/modules", api.GetMicroModules)
		apiRoutes.POST("/activities/:id/complete", api.CompleteActivity)
		apiRoutes.POST("/sync/bulk", api.SyncBulk)
		// Logout: revokes the caller's JWT by adding its JTI to the blocklist
		apiRoutes.POST("/auth/logout", api.LogoutHandler)
	}

	// ---------------------------------------------------------------------------
	// Protected Moderator Routes — requires MODERATOR role minimum
	// ---------------------------------------------------------------------------
	modRoutes := r.Group("/api/moderator")
	modRoutes.Use(api.AuthMiddleware(models.RoleModerator))
	{
		modRoutes.GET("/classes", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Moderator classes data"})
		})
		modRoutes.GET("/roster", api.GetModeratorRoster)
	}

	// ---------------------------------------------------------------------------
	// Protected Admin Routes — requires ADMIN role
	// ---------------------------------------------------------------------------
	adminRoutes := r.Group("/api/admin")
	adminRoutes.Use(api.AuthMiddleware(models.RoleAdmin))
	{
		adminRoutes.GET("/dashboard", api.GetAdminDashboard)
		adminRoutes.GET("/users", api.GetUsers)
		adminRoutes.PUT("/users/:id/role", api.UpdateUserRole)
		adminRoutes.POST("/activities", api.CreateActivity)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,  // time to read request headers + body
		WriteTimeout: 30 * time.Second,  // time to write response
		IdleTimeout:  120 * time.Second, // keep-alive timeout
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		slog.Info("LOG Backend starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown:", "error", err)
	}

	slog.Info("Server exiting")
}
