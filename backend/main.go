package main

import (
	"log-backend/api"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// CORS middleware for local dev
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

	apiRoutes := r.Group("/api")
	{
		apiRoutes.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })
		apiRoutes.GET("/dashboard", api.GetDashboard)
		apiRoutes.GET("/learning-journey", api.GetLearningJourney)
		apiRoutes.GET("/chart-data", api.GetChartData)
	}

	r.Run(":8080")
}
