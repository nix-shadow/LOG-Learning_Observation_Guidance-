package main

import (
	"log-backend/api"
	"log-backend/database"
	"log-backend/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize Database
	database.InitDB()

	r := gin.Default()

	// CORS and Security Headers Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Public Auth Routes
	authRoutes := r.Group("/api/auth")
	{
		authRoutes.POST("/request-otp", api.RequestOTP)
		authRoutes.POST("/verify-otp", api.VerifyOTP)
	}

	// Protected API Routes (Student)
	apiRoutes := r.Group("/api")
	{
		apiRoutes.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })
		apiRoutes.GET("/dashboard", api.GetDashboard)
		apiRoutes.GET("/learning-journey", api.GetLearningJourney)
		apiRoutes.GET("/chart-data", api.GetChartData)
	}

	// Protected Moderator Routes (Teachers)
	modRoutes := r.Group("/api/moderator")
	modRoutes.Use(api.AuthMiddleware(models.RoleModerator))
	{
		modRoutes.GET("/classes", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Moderator classes data"})
		})
	}

	// Protected Admin Routes (Principal/HOD)
	adminRoutes := r.Group("/api/admin")
	adminRoutes.Use(api.AuthMiddleware(models.RoleAdmin))
	{
		adminRoutes.GET("/dashboard", api.GetAdminDashboard)
		adminRoutes.GET("/users", api.GetUsers)
		adminRoutes.PUT("/users/:id/role", api.UpdateUserRole)
		adminRoutes.POST("/activities", api.CreateActivity)
	}

	r.Run(":8080")
}
