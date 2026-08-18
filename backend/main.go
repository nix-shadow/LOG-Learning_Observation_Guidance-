package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/handler"
	"log-backend/internal/repository"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load configuration
	err := godotenv.Load()
	if err != nil {
		slog.Warn("No .env file found or failed to load. Falling back to environment variables.")
	}

	if os.Getenv("JWT_SECRET") == "" {
		slog.Error("JWT_SECRET environment variable is required")
		os.Exit(1)
	}

	// Setup structured JSON logging (slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize Database
	database.InitDB()

	// Initialize Repositories
	userRepo := repository.NewUserRepository(database.DB)
	authRepo := repository.NewAuthRepository(database.DB)
	syncRepo := repository.NewSyncRepository(database.DB)
	courseRepo := repository.NewCourseRepository(database.DB)
	learnerDataRepo := repository.NewLearnerDataRepository(database.DB)
	modRepo := repository.NewModeratorRepository(database.DB)
	progressRepo := repository.NewProgressRepository(database.DB)
	activityRepo := repository.NewActivityRepository(database.DB)
	completionRepo := repository.NewCompletionRepository(database.DB)
	schoolRepo := repository.NewSchoolRepository(database.DB)

	// Initialize Services
	authService := service.NewAuthService(userRepo, authRepo)
	syncService := service.NewSyncService(syncRepo)
	learnerService := service.NewLearnerService(userRepo, activityRepo, progressRepo, learnerDataRepo, completionRepo)
	courseService := service.NewCourseService(courseRepo)
	moderatorService := service.NewModeratorService(modRepo)
	schoolService := service.NewSchoolService(schoolRepo)

	// Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
	syncHandler := handler.NewSyncHandler(syncService)
	learnerHandler := handler.NewLearnerHandler(learnerService, courseService, moderatorService)
	schoolHandler := handler.NewSchoolHandler(schoolService)

	// Use gin.New() + only the Logger and Recovery middlewares we control
	// so we avoid gin.Default()'s panic recovery leaking stack traces to clients.
	r := gin.New()
	// Trust no reverse proxies: ClientIP resolves to the direct peer address so
	// X-Forwarded-For cannot spoof the rate limiter's per-IP key.
	r.SetTrustedProxies(nil)
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
		corsOrigin = "http://localhost:6100"
		slog.Info("CORS_ORIGIN not set, defaulting to http://localhost:6100")
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
	r.Use(handler.RequestIDMiddleware())

	// ---------------------------------------------------------------------------
	// Public Auth Routes — rate limited
	// ---------------------------------------------------------------------------
	authRoutes := r.Group("/api/v1/auth")
	authRoutes.Use(handler.RateLimitMiddleware())
	{
		authRoutes.POST("/request-otp", authHandler.RequestOTP)
		authRoutes.POST("/verify-otp", authHandler.VerifyOTP)
		authRoutes.POST("/forgot-password", authHandler.ForgotPassword)
		authRoutes.POST("/google", authHandler.GoogleAuth)
		authRoutes.POST("/login", authHandler.Login)
	}

	// ---------------------------------------------------------------------------
	// Public Health Probes
	// ---------------------------------------------------------------------------
	r.GET("/api/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })
	r.GET("/healthz", func(c *gin.Context) {
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ready"}) })

	// ---------------------------------------------------------------------------
	// Protected Student Routes — requires valid JWT with STUDENT role minimum
	// ---------------------------------------------------------------------------
	apiRoutes := r.Group("/api/v1")
	apiRoutes.Use(handler.AuthMiddleware(authRepo, userRepo, schoolRepo, domain.RoleStudent))
	{
		apiRoutes.GET("/dashboard", learnerHandler.GetDashboard)
		apiRoutes.GET("/learning-journey", learnerHandler.GetLearningJourney)
		apiRoutes.GET("/chart-data", learnerHandler.GetChartData)
		apiRoutes.GET("/courses", learnerHandler.GetCourses)
		apiRoutes.GET("/activities/:id/modules", learnerHandler.GetMicroModules)
		apiRoutes.POST("/activities/:id/complete", learnerHandler.CompleteActivity)
		apiRoutes.POST("/sync/bulk", syncHandler.SyncBulk)
		// Logout: revokes the caller's JWT by adding its JTI to the blocklist
		apiRoutes.POST("/auth/logout", authHandler.LogoutHandler)
		apiRoutes.PUT("/auth/password", authHandler.UpdatePassword)
		apiRoutes.POST("/auth/logout-all", schoolHandler.LogoutAll)
		// Announcements — read-only for learners
		apiRoutes.GET("/announcements", schoolHandler.ListAnnouncements)
		// Assignments — learners see their class assignments and submit answers
		apiRoutes.GET("/assignments", schoolHandler.ListMyAssignments)
		apiRoutes.POST("/assignments/:assignment_id/submit", schoolHandler.SubmitAssignment)
	}

	// ---------------------------------------------------------------------------
	// Protected Moderator Routes — requires MODERATOR role minimum
	// ---------------------------------------------------------------------------
	modRoutes := r.Group("/api/v1/moderator")
	modRoutes.Use(handler.AuthMiddleware(authRepo, userRepo, schoolRepo, domain.RoleModerator))
	{
		modRoutes.GET("/roster", learnerHandler.GetModeratorRoster)
		modRoutes.GET("/classes", schoolHandler.ListMyClasses)
		modRoutes.GET("/classes/:id/assignments", schoolHandler.ListAssignmentsForClass)
		modRoutes.POST("/classes/:id/assignments", schoolHandler.CreateAssignment)
		modRoutes.GET("/classes/:id/assignments/:assignment_id/submissions", schoolHandler.SubmissionsForAssignment)
		// Teachers may publish announcements too
		modRoutes.POST("/announcements", schoolHandler.CreateAnnouncement)
	}

	// ---------------------------------------------------------------------------
	// Protected Admin Routes — requires ADMIN role
	// ---------------------------------------------------------------------------
	adminRoutes := r.Group("/api/v1/admin")
	adminRoutes.Use(handler.AuthMiddleware(authRepo, userRepo, schoolRepo, domain.RoleAdmin))
	{
		adminRoutes.GET("/dashboard", handler.GetAdminDashboard)
		adminRoutes.GET("/users", handler.GetUsers)
		adminRoutes.PUT("/users/:id/role", handler.UpdateUserRole)
		adminRoutes.POST("/activities", handler.CreateActivity)
		adminRoutes.POST("/classes", schoolHandler.CreateClass)
		adminRoutes.GET("/classes", schoolHandler.ListClasses)
		adminRoutes.GET("/classes/:id/roster", schoolHandler.ClassRoster)
		adminRoutes.POST("/classes/:id/enroll", schoolHandler.EnrollStudents)
		adminRoutes.DELETE("/classes/:id/enroll/:user_id", schoolHandler.UnenrollStudent)
		adminRoutes.POST("/announcements", schoolHandler.CreateAnnouncement)
		adminRoutes.GET("/audit-log", schoolHandler.ListAuditLog)
		adminRoutes.GET("/export/students.csv", schoolHandler.ExportStudentsCSV)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "6101"
		slog.Info("PORT not set, defaulting to 6101")
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
