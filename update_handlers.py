import re

with open('backend/api/handlers.go', 'r') as f:
    content = f.read()

# 1. Update GetDashboard
get_dashboard_orig = """	var activities []models.Activity
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
		"activities":   activities,"""

get_dashboard_new = """	var dbActivities []models.Activity
	var observations []models.Observation
	var guidance []models.Guidance

	// Resolve the authenticated learner — never fall back to another user's data
	if err := database.DB.First(&user, "id = ?", learnerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Learner account not found"})
		return
	}

	database.DB.Order("`order` asc").Find(&dbActivities)

	var learnerActs []models.LearnerActivity
	database.DB.Where("learner_id = ?", learnerID).Find(&learnerActs)
	statusMap := make(map[string]string)
	for _, la := range learnerActs {
		statusMap[la.ActivityID] = la.Status
	}

	type ActivityResponse struct {
		models.Activity
		Status string `json:"status"`
	}
	var activities []ActivityResponse
	for _, act := range dbActivities {
		status := "Pending"
		if s, ok := statusMap[act.ID]; ok {
			status = s
		}
		activities = append(activities, ActivityResponse{
			Activity: act,
			Status:   status,
		})
	}

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
		"activities":   activities,"""

content = content.replace(get_dashboard_orig, get_dashboard_new)

# 2. Update GetLearningJourney
get_learning_journey_orig = """func GetLearningJourney(c *gin.Context) {
	var activities []models.Activity
	database.DB.Order("`order` asc").Find(&activities)
	c.JSON(http.StatusOK, gin.H{"activities": activities})
}"""

get_learning_journey_new = """func GetLearningJourney(c *gin.Context) {
	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	var dbActivities []models.Activity
	database.DB.Order("`order` asc").Find(&dbActivities)

	var learnerActs []models.LearnerActivity
	database.DB.Where("learner_id = ?", learnerID).Find(&learnerActs)
	statusMap := make(map[string]string)
	for _, la := range learnerActs {
		statusMap[la.ActivityID] = la.Status
	}

	type ActivityResponse struct {
		models.Activity
		Status string `json:"status"`
	}
	var activities []ActivityResponse
	for _, act := range dbActivities {
		status := "Pending"
		if s, ok := statusMap[act.ID]; ok {
			status = s
		}
		activities = append(activities, ActivityResponse{
			Activity: act,
			Status:   status,
		})
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}"""
content = content.replace(get_learning_journey_orig, get_learning_journey_new)

# 3. Update GetMicroModules
get_micromodules_orig = """	var activity models.Activity
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
	})"""

get_micromodules_new = """	learnerID := "user-123"
	if uid, exists := c.Get("userID"); exists && uid.(string) != "" {
		learnerID = uid.(string)
	}

	var activity models.Activity
	if err := database.DB.First(&activity, "id = ?", actID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	var learnerAct models.LearnerActivity
	status := "Pending"
	if err := database.DB.First(&learnerAct, "learner_id = ? AND activity_id = ?", learnerID, actID).Error; err == nil {
		status = learnerAct.Status
	}

	type ActivityResponse struct {
		models.Activity
		Status string `json:"status"`
	}

	var modules []models.MicroModule
	database.DB.Where("activity_id = ?", actID).Order("`order` asc").Find(&modules)

	c.JSON(http.StatusOK, gin.H{
		"activity": ActivityResponse{
			Activity: activity,
			Status:   status,
		},
		"modules":  modules,
		"total":    len(modules),
	})"""
content = content.replace(get_micromodules_orig, get_micromodules_new)

# 4. Update CompleteActivity
complete_activity_orig = """		// 1. Update Activity Status
		var activity models.Activity
		if err := tx.First(&activity, "id = ?", actID).Error; err != nil {
			return fmt.Errorf("activity not found: %s", actID)
		}
		activity.Status = "Completed"
		if err := tx.Save(&activity).Error; err != nil {
			return fmt.Errorf("failed to update activity: %w", err)
		}
		resultActivity = activity"""

complete_activity_new = """		// 1. Update Activity Status (via LearnerActivity)
		var activity models.Activity
		if err := tx.First(&activity, "id = ?", actID).Error; err != nil {
			return fmt.Errorf("activity not found: %s", actID)
		}
		resultActivity = activity

		var learnerAct models.LearnerActivity
		if err := tx.First(&learnerAct, "learner_id = ? AND activity_id = ?", learnerID, actID).Error; err != nil {
			learnerAct = models.LearnerActivity{
				LearnerID:   learnerID,
				ActivityID:  actID,
				Status:      "Completed",
				CompletedAt: time.Now(),
				Score:       100.0,
			}
			if err := tx.Create(&learnerAct).Error; err != nil {
				return fmt.Errorf("failed to create learner activity: %w", err)
			}
		} else {
			learnerAct.Status = "Completed"
			learnerAct.CompletedAt = time.Now()
			if err := tx.Save(&learnerAct).Error; err != nil {
				return fmt.Errorf("failed to update learner activity: %w", err)
			}
		}"""
content = content.replace(complete_activity_orig, complete_activity_new)

# 5. Update SyncBulk
syncbulk_orig = """						act.Status = "Completed"
						tx.Save(&act)"""

syncbulk_new = """						var learnerAct models.LearnerActivity
						if err := tx.First(&learnerAct, "learner_id = ? AND activity_id = ?", callerID, actID).Error; err != nil {
							learnerAct = models.LearnerActivity{
								LearnerID:   callerID,
								ActivityID:  actID,
								Status:      "Completed",
								CompletedAt: time.Now(),
								Score:       100.0,
							}
							tx.Create(&learnerAct)
						} else {
							learnerAct.Status = "Completed"
							learnerAct.CompletedAt = time.Now()
							tx.Save(&learnerAct)
						}"""
content = content.replace(syncbulk_orig, syncbulk_new)

with open('backend/api/handlers.go', 'w') as f:
    f.write(content)
