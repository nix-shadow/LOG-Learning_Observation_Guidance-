package main

import (
	"log-backend/api"
	"log-backend/models"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	authRoutes := r.Group("/api/auth")
	{
		authRoutes.POST("/request-otp", api.RequestOTP)
		authRoutes.POST("/verify-otp", api.VerifyOTP)
		authRoutes.POST("/google", api.GoogleLoginMock)
		authRoutes.POST("/forgot-password", api.ForgotPassword)
	}

	apiRoutes := r.Group("/api")
	{
		apiRoutes.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })
		apiRoutes.GET("/dashboard", api.GetDashboard)
		apiRoutes.GET("/learning-journey", api.GetLearningJourney)
		apiRoutes.GET("/chart-data", api.GetChartData)
	}

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
