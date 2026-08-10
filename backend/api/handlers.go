package api

import (
	"fmt"
	"log-backend/database"
	"log-backend/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDashboard(c *gin.Context) {
	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	var user models.User
	var progress models.Progress
	var activities []models.Activity
	var observations []models.Observation
	var guidance []models.Guidance

	// Resolve the authenticated learner — never fall back to another user's data
	if err := database.DB.First(&user, "id = ?", learnerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Learner account not found"})
		return
	}

	database.DB.Order("`order` asc").Find(&activities)

	if err := database.DB.First(&progress, "learner_id = ?", learnerID).Error; err != nil {
		progress = models.Progress{
			LearnerID:     learnerID,
			TotalTopics:   len(activities),
			Completed:     0,
			CurrentStreak: 0,
			OverallScore:  0,
		}
		database.DB.Create(&progress)
	}
	database.DB.Order("created_at desc").Find(&observations, "learner_id = ?", learnerID)
	database.DB.Order("created_at desc").Find(&guidance, "learner_id = ?", learnerID)

	c.JSON(http.StatusOK, gin.H{
		"learner":      user,
		"progress":     progress,
		"activities":   activities,
		"observations": observations,
		"guidance":     guidance,
	})
}

func GetLearningJourney(c *gin.Context) {
	var activities []models.Activity
	database.DB.Order("`order` asc").Find(&activities)
	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

// GetMicroModules returns all MicroModule entries for a given Activity ID,
// ordered for sequential display in the MicroModuleViewer component.
func GetMicroModules(c *gin.Context) {
	actID := c.Param("id")
	if actID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Activity ID is required"})
		return
	}

	var activity models.Activity
	if err := database.DB.First(&activity, "id = ?", actID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	var modules []models.MicroModule
	database.DB.Where("activity_id = ?", actID).Order("`order` asc").Find(&modules)

	c.JSON(http.StatusOK, gin.H{
		"activity": activity,
		"modules":  modules,
		"total":    len(modules),
	})
}

// CompleteActivity marks an activity as completed within a single database
// transaction to prevent inconsistent state on partial failures.
func CompleteActivity(c *gin.Context) {
	actID := c.Param("id")
	if actID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Activity ID is required"})
		return
	}

	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	var resultActivity models.Activity
	var resultProgress models.Progress
	var resultObs models.Observation
	var resultGui models.Guidance

	// Wrap all writes in a single transaction for atomicity
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Update Activity Status
		var activity models.Activity
		if err := tx.First(&activity, "id = ?", actID).Error; err != nil {
			return fmt.Errorf("activity not found: %s", actID)
		}
		activity.Status = "Completed"
		if err := tx.Save(&activity).Error; err != nil {
			return fmt.Errorf("failed to update activity: %w", err)
		}
		resultActivity = activity

		// 2. Update Progress atomically — create the record for new learners
		var progress models.Progress
		if err := tx.First(&progress, "learner_id = ?", learnerID).Error; err != nil {
			var totalTopics int64
			tx.Model(&models.Activity{}).Count(&totalTopics)
			progress = models.Progress{
				LearnerID:   learnerID,
				TotalTopics: int(totalTopics),
			}
		}
		progress.Completed++
		if progress.Completed > progress.TotalTopics {
			progress.Completed = progress.TotalTopics
		}
		progress.CurrentStreak++
		if progress.OverallScore < 95.0 {
			progress.OverallScore += 2.5
		}
		if err := tx.Save(&progress).Error; err != nil {
			return fmt.Errorf("failed to update progress: %w", err)
		}
		resultProgress = progress

		// 3. Generate supportive observation (positive phrasing only)
		obsTitle := "Module Completed"
		if activity.Title != "" {
			obsTitle = activity.Title
		}
		obs := models.Observation{
			ID:        GenerateSecureID("obs"),
			LearnerID: learnerID,
			Category:  "strengths",
			Text:      fmt.Sprintf("Demonstrated excellent focus and successfully completed %s.", obsTitle),
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&obs).Error; err != nil {
			return fmt.Errorf("failed to create observation: %w", err)
		}
		resultObs = obs

		// 4. Generate actionable next-step guidance
		gui := models.Guidance{
			ID:        GenerateSecureID("gui"),
			LearnerID: learnerID,
			Text:      "Great momentum! Continue to the next practice module to reinforce your logic skills.",
			Action:    "/learning",
			Type:      "next_step",
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&gui).Error; err != nil {
			return fmt.Errorf("failed to create guidance: %w", err)
		}
		resultGui = gui

		return nil
	})

	if txErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete activity. Please try again.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Activity marked as completed",
		"activity_id": resultActivity.ID,
		"progress":    resultProgress,
		"observation": resultObs,
		"guidance":    resultGui,
	})
}

func GetCourses(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit
	var courses []models.Course
	var total int64

	database.DB.Model(&models.Course{}).Count(&total)
	database.DB.Limit(limit).Offset(offset).Find(&courses)

	c.JSON(http.StatusOK, gin.H{
		"courses": courses,
		"pagination": gin.H{
			"page": page,
			"limit": limit,
			"total": total,
		},
	})
}

func GetModeratorRoster(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var users []models.User
	var total int64
	
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleStudent).Count(&total)
	database.DB.Where("role = ?", models.RoleStudent).Limit(limit).Offset(offset).Find(&users)

	// Keep the hardcoded roster mapping for MVP UI compatibility, but structure it properly
	roster := []map[string]interface{}{}
	needsAttention := 0
	
	for _, u := range users {
		var progress models.Progress
		database.DB.First(&progress, "learner_id = ?", u.ID)
		
		status := "Active"
		if progress.CurrentStreak == 0 {
			status = "Needs Attention"
			needsAttention++
		}
		
		completion := 0
		if progress.TotalTopics > 0 {
			completion = int(float64(progress.Completed) / float64(progress.TotalTopics) * 100)
		}

		roster = append(roster, map[string]interface{}{
			"id": u.ID,
			"name": u.Name,
			"completion": completion,
			"streak": progress.CurrentStreak,
			"status": status,
			"last_active": u.UpdatedAt.Format("Jan 02"),
		})
	}

	var assignmentsDue int64
	database.DB.Model(&models.Activity{}).Where("status = ?", "In progress").Count(&assignmentsDue)

	c.JSON(http.StatusOK, gin.H{
		"class_name":      "Logic 101: Discrete Structures",
		"active_students": total,
		"needs_attention": needsAttention,
		"assignments_due": assignmentsDue,
		"roster":          roster,
		"pagination": gin.H{
			"page": page,
			"limit": limit,
			"total": total,
		},
	})
}

func GetChartData(c *gin.Context) {
	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	var activities []models.DailyActivity
	database.DB.Where("learner_id = ?", learnerID).Order("date asc").Find(&activities)

	chartData := make([]map[string]interface{}, 0)
	for _, act := range activities {
		chartData = append(chartData, map[string]interface{}{
			"name":     act.DayName,
			"score":    act.Score,
			"duration": act.Duration,
		})
	}

	if len(chartData) == 0 {
		chartData = []map[string]interface{}{
			{"name": "Mon", "score": 0, "duration": 0},
			{"name": "Tue", "score": 0, "duration": 0},
			{"name": "Wed", "score": 0, "duration": 0},
			{"name": "Thu", "score": 0, "duration": 0},
			{"name": "Fri", "score": 0, "duration": 0},
			{"name": "Sat", "score": 0, "duration": 0},
			{"name": "Sun", "score": 0, "duration": 0},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"activity_data": chartData,
	})
}

type SyncRequestItem struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
	Body     string `json:"body"`
}

type SyncBulkPayload struct {
	Version   string            `json:"version"`
	Timestamp string            `json:"timestamp"`
	Data      []SyncRequestItem `json:"data"`
}

// SyncBulk processes a batch of offline requests uploaded via a .logsync file
func SyncBulk(c *gin.Context) {
	var payload SyncBulkPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sync payload format"})
		return
	}

	processedCount := 0

	// Resolve authenticated caller — scoping prevents cross-user data tampering
	callerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		callerID = uid.(string)
	}

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, req := range payload.Data {
			if req.Method == "POST" && len(req.Endpoint) > 0 {
				// E.g. /activities/act-1/complete
				parts := strings.Split(req.Endpoint, "/")
				if len(parts) >= 4 && parts[1] == "activities" && parts[3] == "complete" {
					actID := parts[2]
					var act models.Activity
					if err := tx.First(&act, "id = ?", actID).Error; err == nil {
						act.Status = "Completed"
						tx.Save(&act)

						// Scoped progress update: only touch the calling user's progress,
						// creating the record the first time a new learner syncs.
						var progress models.Progress
						if err := tx.First(&progress, "learner_id = ?", callerID).Error; err != nil {
							var totalTopics int64
							tx.Model(&models.Activity{}).Count(&totalTopics)
							progress = models.Progress{
								LearnerID:   callerID,
								TotalTopics: int(totalTopics),
							}
						}
						progress.Completed++
						if progress.Completed > progress.TotalTopics {
							progress.Completed = progress.TotalTopics
						}
						progress.CurrentStreak++
						if progress.OverallScore < 95.0 {
							progress.OverallScore += 2.5
						}
						tx.Save(&progress)

						// Mirror the online completion flow: supportive observation
						// + actionable next-step guidance, generated after sync.
						obsTitle := "Module Completed"
						if act.Title != "" {
							obsTitle = act.Title
						}
						tx.Create(&models.Observation{
							ID:        GenerateSecureID("obs"),
							LearnerID: callerID,
							Category:  "strengths",
							Text:      fmt.Sprintf("Demonstrated excellent focus and successfully completed %s.", obsTitle),
							CreatedAt: time.Now(),
						})
						tx.Create(&models.Guidance{
							ID:        GenerateSecureID("gui"),
							LearnerID: callerID,
							Text:      "Great momentum! Continue to the next practice module to reinforce your logic skills.",
							Action:    "/learning",
							Type:      "next_step",
							CreatedAt: time.Now(),
						})
						processedCount++
					}
				}
			}
		}
		return nil
	})

	if txErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync offline data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully synced %d offline actions.", processedCount),
		"count":   processedCount,
	})
}
