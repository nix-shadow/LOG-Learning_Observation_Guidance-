package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetAdminDashboard(c *gin.Context) {
	var totalUsers int64
	var totalActivities int64
	var totalCompletions int64
	var activeDaily int64

	database.DB.Model(&domain.User{}).Count(&totalUsers)
	database.DB.Model(&domain.Activity{}).Count(&totalActivities)
	database.DB.Model(&domain.Progress{}).Select("COALESCE(SUM(completed), 0)").Scan(&totalCompletions)
	// Active daily = learners with real completion activity in the last 24
	// hours. users.updated_at only moves on password/role changes, so counting
	// users by it made this metric permanently ~0 — LearnerActivity.completed_at
	// is the honest signal.
	database.DB.Model(&domain.LearnerActivity{}).
		Where("completed_at > ?", time.Now().Add(-24*time.Hour)).
		Distinct("learner_id").
		Count(&activeDaily)

	var recentUsers []domain.User
	database.DB.Order("created_at desc").Limit(5).Find(&recentUsers)

	c.JSON(http.StatusOK, gin.H{
		"analytics": domain.SystemAnalytics{
			TotalUsers:       int(totalUsers),
			ActiveDaily:      int(activeDaily),
			TotalCompletions: int(totalCompletions),
		},
		"recent_users": recentUsers,
	})
}

func GetUsers(c *gin.Context) {
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

	var users []domain.User
	var total int64

	// Deterministic ordering — SQLite's default (rowid) order made the first
	// page arbitrary, and the admin UI's enrollment form relies on this list.
	database.DB.Model(&domain.User{}).Count(&total)
	database.DB.Order("created_at desc").Limit(limit).Offset(offset).Find(&users)

	// Consent status (WP-0.1): principals need to see which learners have
	// guardian consent before relying on their participation. Additive field —
	// null when absent, never a fabricated value.
	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	var consentRows []domain.ConsentRecord
	database.DB.Where("user_id IN ? AND consent_type = ? AND status = ?", userIDs, domain.ConsentTypeGuardian, domain.ConsentStatusGranted).
		Find(&consentRows)
	consentByUser := make(map[string]domain.ConsentRecord, len(consentRows))
	for _, rec := range consentRows {
		if prev, ok := consentByUser[rec.UserID]; !ok || rec.GrantedAt.After(prev.GrantedAt) {
			consentByUser[rec.UserID] = rec
		}
	}

	type userWithConsent struct {
		domain.User
		Consent *domain.ConsentRecord `json:"consent"`
	}
	response := make([]userWithConsent, 0, len(users))
	for _, u := range users {
		var consent *domain.ConsentRecord
		if rec, ok := consentByUser[u.ID]; ok {
			recCopy := rec
			consent = &recCopy
		}
		response = append(response, userWithConsent{User: u, Consent: consent})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": response,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// UpdateUserRole changes a user's role. The check-then-act last-admin guard,
// the role write, and the audit entry run inside ONE transaction so two
// concurrent demotions can never leave a school with zero admins, and a
// failed write never reports success.
func UpdateUserRole(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Role domain.Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid role")
		return
	}

	// Validate role is one of the defined constants — reject any arbitrary string
	switch req.Role {
	case domain.RoleStudent, domain.RoleModerator, domain.RoleAdmin:
		// valid — proceed
	default:
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid role value. Must be STUDENT, MODERATOR, or ADMIN.")
		return
	}

	actor, _ := c.Get("userID")
	roleChange := database.DB.Transaction(func(tx *gorm.DB) error {
		var user domain.User
		if err := tx.First(&user, "id = ?", id).Error; err != nil {
			return errUserNotFound
		}

		// Never demote the last remaining ADMIN — a school with zero principals
		// has no recovery path (nobody can promote a new one). Re-counted inside
		// the transaction so a concurrent demotion cannot pass the guard.
		if user.Role == domain.RoleAdmin && req.Role != domain.RoleAdmin {
			var adminCount int64
			if err := tx.Model(&domain.User{}).
				Where("role = ? AND id <> ?", domain.RoleAdmin, id).
				Count(&adminCount).Error; err != nil {
				return err
			}
			if adminCount < 1 {
				return errLastAdmin
			}
		}

		user.Role = req.Role
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// Append-only audit trail for sensitive privilege changes — written in
		// the same transaction as the mutation, so the trail cannot drift.
		if err := tx.Create(&domain.AuditLog{
			UserID:    actor.(string),
			Action:    "user.role_change",
			Detail:    id + " -> " + string(req.Role),
			IP:        c.ClientIP(),
			CreatedAt: time.Now(),
		}).Error; err != nil {
			return err
		}
		return nil
	})

	switch roleChange {
	case nil:
		var user domain.User
		database.DB.First(&user, "id = ?", id)
		c.JSON(http.StatusOK, gin.H{"message": "Role updated", "user": user})
	case errLastAdmin:
		RespondError(c, http.StatusBadRequest, "Bad Request", "Cannot demote the last admin. Promote another user to ADMIN first.")
	case errUserNotFound:
		RespondError(c, http.StatusNotFound, "Not Found", "User not found")
	default:
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to update role")
	}
}

// Sentinel errors so the transaction can report exactly why it rolled back.
var (
	errLastAdmin    = errors.New("last admin")
	errUserNotFound = errors.New("user not found")
)

// CreateActivityRequest is a strict DTO that prevents clients from injecting
// server-managed fields like ID, CreatedAt, or Order.
type CreateActivityRequest struct {
	Title         string `json:"title" binding:"required,min=3,max=200"`
	Description   string `json:"description" binding:"required"`
	Topic         string `json:"topic" binding:"required"`
	Difficulty    string `json:"difficulty" binding:"required,oneof=Beginner Intermediate Advanced"`
	Prerequisites string `json:"prerequisites"`
	ContentJSON   string `json:"content_json"`
}

func CreateActivity(c *gin.Context) {
	var req CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	// Count existing activities to auto-assign display order
	var count int64
	database.DB.Model(&domain.Activity{}).Count(&count)

	act := domain.Activity{
		ID:            service.GenerateSecureID("act"), // Server-generated ID
		Title:         req.Title,
		Description:   req.Description,
		Topic:         req.Topic,
		Difficulty:    req.Difficulty,
		Prerequisites: req.Prerequisites,
		ContentJSON:   req.ContentJSON,
		Order:         int(count) + 1,
		CreatedAt:     time.Now(),
	}

	if err := database.DB.Create(&act).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to create activity")
		return
	}
	actor, _ := c.Get("userID")
	database.DB.Create(&domain.AuditLog{
		UserID:    actor.(string),
		Action:    "activity.create",
		Detail:    act.ID + " " + act.Title,
		IP:        c.ClientIP(),
		CreatedAt: time.Now(),
	})
	c.JSON(http.StatusCreated, act)
}
