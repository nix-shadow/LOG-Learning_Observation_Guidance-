package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// PrivacyHandler serves the WP-0.1 privacy endpoints: consent recording and
// status, personal-data export, and account erasure. All routes require an
// authenticated caller (student minimum) and are rate limited per IP in
// main.go — the same hardening as the auth routes.
type PrivacyHandler struct {
	privacyService service.PrivacyService
}

func NewPrivacyHandler(privacyService service.PrivacyService) *PrivacyHandler {
	return &PrivacyHandler{privacyService: privacyService}
}

// ConsentRequest is a strict DTO: server-managed fields (ID, status, GrantedAt)
// cannot be injected by the client, and strings are length-capped.
type ConsentRequest struct {
	ConsentType     string `json:"consent_type" binding:"required"`
	Version         string `json:"version" binding:"required"`
	GrantedBy       string `json:"granted_by" binding:"required"`
	GuardianName    string `json:"guardian_name" binding:"omitempty,max=120"`
	GuardianContact string `json:"guardian_contact" binding:"omitempty,max=120"`
	Language        string `json:"language" binding:"required"`
	Source          string `json:"source" binding:"omitempty,max=40"`
	DisclosureHash  string `json:"disclosure_hash" binding:"omitempty"`
}

// RecordConsent stores or refreshes the caller's consent evidence.
// POST /api/v1/me/consent
func (h *PrivacyHandler) RecordConsent(c *gin.Context) {
	var req ConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid consent payload: "+err.Error())
		return
	}

	userID, _ := c.Get("userID")
	record, err := h.privacyService.RecordConsent(c.Request.Context(), userID.(string), service.ConsentInput{
		ConsentType:     req.ConsentType,
		Version:         req.Version,
		GrantedBy:       req.GrantedBy,
		GuardianName:    req.GuardianName,
		GuardianContact: req.GuardianContact,
		Language:        req.Language,
		Source:          req.Source,
		DisclosureHash:  req.DisclosureHash,
		IP:              c.ClientIP(),
	})
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	database.DB.Create(&domain.AuditLog{
		UserID:    userID.(string),
		Action:    "privacy.consent_granted",
		Detail:    req.ConsentType + " v" + req.Version,
		IP:        c.ClientIP(),
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusCreated, record)
}

// GetMyConsent reports the caller's consent history plus the policy version
// and retention commitments that apply to their data.
// GET /api/v1/me/consent
func (h *PrivacyHandler) GetMyConsent(c *gin.Context) {
	userID, _ := c.Get("userID")
	records, err := h.privacyService.GetConsents(c.Request.Context(), userID.(string))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to load consent records")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"consent":  records,
		"required": domain.ConsentTypeGuardian,
		"policy": gin.H{
			"version": domain.PolicyVersion,
			"retention": gin.H{
				"inactive_account_years": domain.InactiveAccountRetentionYears,
				"audit_log_years":        domain.AuditLogRetentionYears,
				"offline_queue_days":     domain.QueuedDataRetentionDays,
			},
			// Research round (WP-0.1): the notice text now states explicitly
			// that LOG never discloses learner data to third parties — the
			// API surface mirrors that commitment instead of leaving it
			// implied. No ad-tech, no brokers, no data for sale.
			"sharing": gin.H{
				"third_party_disclosure": false,
			},
		},
	})
}

// ExportMyData returns every table that holds the caller's own data in a
// portable, self-describing JSON envelope (Google Takeout-style). Derived or
// inferred analytics are excluded — only provided + observed data is portable.
// GET /api/v1/me/export
func (h *PrivacyHandler) ExportMyData(c *gin.Context) {
	userID, _ := c.Get("userID")
	bundle, err := h.privacyService.ExportData(c.Request.Context(), userID.(string))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to gather export data")
		return
	}

	actor, _ := c.Get("userID")
	database.DB.Create(&domain.AuditLog{
		UserID:    actor.(string),
		Action:    "privacy.data_export",
		Detail:    "schema_v1",
		IP:        c.ClientIP(),
		CreatedAt: time.Now(),
	})

	// The envelope makes the export self-describing: when it is re-imported
	// anywhere, the recipient knows the policy version and retention rules
	// that applied at export time.
	envelope := gin.H{
		"exported_at":    time.Now().UTC().Format(time.RFC3339),
		"schema_version": 1,
		"policy_version": domain.PolicyVersion,
		"retention": gin.H{
			"inactive_account_years": domain.InactiveAccountRetentionYears,
			"audit_log_years":        domain.AuditLogRetentionYears,
			"offline_queue_days":     domain.QueuedDataRetentionDays,
		},
		"sharing": gin.H{
			"third_party_disclosure": false,
		},
		"user":               bundle.User,
		"consent":            bundle.Consents,
		"progress":           bundle.Progress,
		"learner_activities": bundle.LearnerActivities,
		"observations":       bundle.Observations,
		"guidance":           bundle.Guidance,
		"daily_activities":   bundle.DailyActivities,
		"classes":            bundle.Classes,
		"submissions":        bundle.Submissions,
		"audit_log":          bundle.AuditLog,
		"notice":             "This export contains the data you provided to LOG or that was observed about you. Derived analytics are not included. LOG never discloses learner data to third parties. Retention: learner data is kept at most " + fmt.Sprintf("%d", domain.InactiveAccountRetentionYears) + " years after last activity; audit records at most " + fmt.Sprintf("%d", domain.AuditLogRetentionYears) + " years.",
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"log-data-export-%s.json\"", time.Now().Format("2006-01-02")))
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, envelope)
}

// DeleteAccountRequest requires an explicit confirmation so a stray client
// replay cannot erase an account.
type DeleteAccountRequest struct {
	Confirm string `json:"confirm" binding:"required"`
}

// DeleteAccount erases the caller's data per the erasure data map and
// hard-deletes the account. The anonymized audit trail is written atomically
// with the erasure inside the transaction.
// DELETE /api/v1/me
func (h *PrivacyHandler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Invalid request body")
		return
	}
	if req.Confirm != "DELETE" {
		RespondError(c, http.StatusBadRequest, "Bad Request", "Type DELETE to confirm account deletion")
		return
	}

	userID, _ := c.Get("userID")
	if err := h.privacyService.DeleteAccount(c.Request.Context(), userID.(string)); err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Failed to delete account")
		return
	}

	// WP-0.1 research (SQLite forensics): DELETE alone leaves recoverable rows
	// in freelist pages and the WAL file. ScrubDeletedData checkpoints the WAL
	// (TRUNCATE) and VACUUMs so the erasure is real, not just logical. This is
	// best-effort: the erasure already committed; a failed scrub is logged and
	// the caller still gets 200 (the runbook covers the manual follow-up).
	// The returned report is logged as verification evidence so an operator
	// can confirm the wipe actually shrank the recoverable surface.
	report, err := h.privacyService.ScrubDeletedData(c.Request.Context())
	if err != nil {
		slog.Warn("Post-erasure VACUUM/checkpoint failed (manual scrub may be needed):", "error", err.Error())
	} else {
		slog.Info("Post-erasure scrub verified",
			"freelist_before", report.FreelistBefore,
			"freelist_after", report.FreelistAfter,
			"wal_frames", report.WALFrames,
			"wal_bytes", report.WALBytes,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Account deleted",
		"retention": "Learner data has been erased. Anonymized audit records may remain for " + fmt.Sprintf("%d", domain.AuditLogRetentionYears) + " years.",
	})
}

// PurgeNow runs the retention purge on demand (admin maintenance endpoint).
// The evidence report is returned to the caller; each erasure also writes its
// own anonymized audit row, so the run is fully observable.
// POST /api/v1/admin/maintenance/purge
func (h *PrivacyHandler) PurgeNow(c *gin.Context) {
	report, err := h.privacyService.PurgeExpiredData(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Internal Server Error", "Retention purge failed: "+err.Error())
		return
	}
	slog.Info("Manual retention purge",
		"by", c.GetString("userID"),
		"users_purged", report.UsersPurged,
		"audit_rows_purged", report.AuditRowsPurged,
	)
	c.JSON(http.StatusOK, gin.H{
		"message": "Retention purge complete",
		"report":  report,
	})
}
