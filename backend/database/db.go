package database

import (
	"log"
	"log-backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("log.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto Migrate
	err = DB.AutoMigrate(
		&models.User{},
		&models.OTPRecord{},
		&models.Activity{},
		&models.Progress{},
		&models.Observation{},
		&models.Guidance{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	seedData()
}

func seedData() {
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		// Seed Users
		users := []models.User{
			{ID: "admin-1", Name: "Principal Skinner", Email: "admin@log.edu", Phone: "1000000000", Role: models.RoleAdmin, IsVerified: true},
			{ID: "mod-1", Name: "Teacher Edna", Email: "teacher@log.edu", Phone: "2000000000", Role: models.RoleModerator, IsVerified: true},
			{ID: "user-123", Name: "Aisha Student", Email: "aisha@example.com", Phone: "+9779800000000", Role: models.RoleStudent, IsVerified: true},
		}
		for _, u := range users {
			DB.Create(&u)
		}

		// Seed Progress & Acts
		DB.Create(&models.Progress{LearnerID: "user-123", TotalTopics: 10, Completed: 2, CurrentStreak: 3, OverallScore: 85.5})
		acts := []models.Activity{
			{ID: "act-1", Title: "Introduction to Logic", Description: "Basic concepts.", Status: "Completed", Topic: "Logic", Order: 1},
			{ID: "act-2", Title: "Boolean Algebra", Description: "AND, OR, NOT.", Status: "In progress", Topic: "Logic", Order: 2},
		}
		for _, a := range acts {
			DB.Create(&a)
		}

		obs := []models.Observation{
			{ID: "obs-1", LearnerID: "user-123", Category: "strengths", Text: "Strong grasp of Boolean Algebra."},
			{ID: "obs-2", LearnerID: "user-123", Category: "consistency", Text: "Studying consistently for 3 days."},
		}
		for _, o := range obs {
			DB.Create(&o)
		}

		gui := []models.Guidance{
			{ID: "gui-1", LearnerID: "user-123", Type: "next_step", Text: "Continue Boolean Algebra.", Action: "/learning/act-2"},
		}
		for _, g := range gui {
			DB.Create(&g)
		}
	}
}
