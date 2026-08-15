package database

import (
	"log/slog"
	"os"
	"time"
	"log-backend/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	var err error

	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		slog.Error("Failed to create data directory:", "error", err)
		os.Exit(1)
	}

	DB, err = gorm.Open(sqlite.Open("data/log.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		slog.Error("Failed to connect to database:", "error", err)
		os.Exit(1)
	}

	// Auto Migrate
	err = DB.AutoMigrate(
		&domain.User{},
		&domain.OTPRecord{},
		&domain.Activity{},
		&domain.LearnerActivity{},
		&domain.Progress{},
		&domain.Observation{},
		&domain.Guidance{},
		&domain.Course{},
		&domain.DailyActivity{},
		&domain.MicroModule{},
		&domain.TokenBlocklist{},
	)
	if err != nil {
		slog.Error("Failed to migrate database:", "error", err)
		os.Exit(1)
	}

	// Purge expired blocklist entries on startup to keep the table lean
	DB.Where("expires_at < ?", time.Now()).Delete(&domain.TokenBlocklist{})

	seedData()
}

func seedData() {
	var count int64
	DB.Model(&domain.User{}).Count(&count)
	if count == 0 {
		// Seed Users
		users := []domain.User{
			{ID: "admin-1", Name: "Principal Skinner", Email: "admin@log.edu", Phone: "1000000000", Role: domain.RoleAdmin, IsVerified: true},
			{ID: "mod-1", Name: "Teacher Edna", Email: "teacher@log.edu", Phone: "2000000000", Role: domain.RoleModerator, IsVerified: true},
			{ID: "user-123", Name: "Aisha Student", Email: "aisha@example.com", Phone: "+9779800000000", Role: domain.RoleStudent, IsVerified: true},
		}
		for _, u := range users {
			DB.Create(&u)
		}

		// Seed Progress & Acts
		DB.Create(&domain.Progress{LearnerID: "user-123", TotalTopics: 10, Completed: 2, CurrentStreak: 3, OverallScore: 85.5})
		acts := []domain.Activity{
			{ID: "act-1", Title: "Introduction to Logic", Description: "Basic concepts.", Topic: "Logic", Order: 1},
			{ID: "act-2", Title: "Boolean Algebra", Description: "AND, OR, NOT.", Topic: "Logic", Order: 2},
		}
		for _, a := range acts {
			DB.Create(&a)
		}

		learnerActs := []domain.LearnerActivity{
			{LearnerID: "user-123", ActivityID: "act-1", Status: "Completed", CompletedAt: time.Now(), Score: 100},
			{LearnerID: "user-123", ActivityID: "act-2", Status: "In progress", Score: 50},
		}
		for _, la := range learnerActs {
			DB.Create(&la)
		}

		obs := []domain.Observation{
			{ID: "obs-1", LearnerID: "user-123", Category: "strengths", Text: "Strong grasp of Boolean Algebra."},
			{ID: "obs-2", LearnerID: "user-123", Category: "consistency", Text: "Studying consistently for 3 days."},
		}
		for _, o := range obs {
			DB.Create(&o)
		}

		gui := []domain.Guidance{
			{ID: "gui-1", LearnerID: "user-123", Type: "next_step", Text: "Continue Boolean Algebra.", Action: "/learning/act-2"},
		}
		for _, g := range gui {
			DB.Create(&g)
		}
	}

	// Seed Micro-Modules independently of users, so existing databases
	// also receive the bite-sized module content on next startup.
	var mmCount int64
	DB.Model(&domain.MicroModule{}).Count(&mmCount)
	if mmCount == 0 {
		microModules := []domain.MicroModule{
			{ID: "mm-1", ActivityID: "act-1", Title: "What is Logic?", ContentText: "Logic is the study of reasoning. In computing, logic gates make every decision your computer makes — from checking a password to rendering a page.", Order: 1},
			{ID: "mm-2", ActivityID: "act-1", Title: "Truth Values", ContentText: "Every logical statement resolves to one of two values: True or False. Think of them as the two possible answers to any yes/no question.", Order: 2},
			{ID: "mm-3", ActivityID: "act-2", Title: "The AND Operator", ContentText: "AND returns True only when BOTH inputs are True. A strict bouncer: you need an ID AND a ticket to enter.", Order: 1},
			{ID: "mm-4", ActivityID: "act-2", Title: "The OR Operator", ContentText: "OR returns True when AT LEAST ONE input is True. A flexible cashier: you can pay with Cash OR Card.", Order: 2},
			{ID: "mm-5", ActivityID: "act-2", Title: "Your Turn!", ContentText: "Now practice: what does (True AND False) evaluate to? It's False — both inputs must be True. Great job thinking it through!", Order: 3},
		}
		for _, m := range microModules {
			DB.Create(&m)
		}
	}

	// Seed Courses
	var courseCount int64
	DB.Model(&domain.Course{}).Count(&courseCount)
	if courseCount == 0 {
		courses := []domain.Course{
			{ID: "course-1", Title: "Fundamentals of Logic & Gates", Category: "Computer Science", Difficulty: "Beginner", Duration: "3 hours", Rating: 4.9, Enrolled: 1250},
			{ID: "course-2", Title: "Boolean Algebra & Truth Tables", Category: "Computer Science", Difficulty: "Intermediate", Duration: "4 hours", Rating: 4.8, Enrolled: 980},
			{ID: "course-3", Title: "Data Structures & Offline Caching", Category: "Backend", Difficulty: "Advanced", Duration: "6 hours", Rating: 4.9, Enrolled: 740},
			{ID: "course-4", Title: "Modern Frontend & Micro-Animations", Category: "Frontend", Difficulty: "Intermediate", Duration: "5 hours", Rating: 4.7, Enrolled: 1120},
			{ID: "course-5", Title: "UI/UX Accessibility for Low-Bandwidth", Category: "Design", Difficulty: "Beginner", Duration: "2.5 hours", Rating: 5.0, Enrolled: 890},
		}
		for _, c := range courses {
			DB.Create(&c)
		}
	}

	// Seed Daily Activity
	var actCount int64
	DB.Model(&domain.DailyActivity{}).Count(&actCount)
	if actCount == 0 {
		dailyActivities := []domain.DailyActivity{
			{ID: "da-1", LearnerID: "user-123", DayName: "Mon", Score: 65, Duration: 20},
			{ID: "da-2", LearnerID: "user-123", DayName: "Tue", Score: 70, Duration: 25},
			{ID: "da-3", LearnerID: "user-123", DayName: "Wed", Score: 68, Duration: 15},
			{ID: "da-4", LearnerID: "user-123", DayName: "Thu", Score: 75, Duration: 30},
			{ID: "da-5", LearnerID: "user-123", DayName: "Fri", Score: 85, Duration: 45},
			{ID: "da-6", LearnerID: "user-123", DayName: "Sat", Score: 82, Duration: 40},
			{ID: "da-7", LearnerID: "user-123", DayName: "Sun", Score: 88, Duration: 50},
		}
		for _, da := range dailyActivities {
			DB.Create(&da)
		}
	}
}
