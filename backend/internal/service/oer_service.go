package service

import (
	"context"
	"errors"

	"log-backend/internal/domain"
)

// ErrOERInvalidLicense is returned when an import row carries a license that
// is not in the OERAllowedLicenses allowlist (WP-3.1 RC-07). The pipeline
// never guesses a license — it rejects the row with a human reason.
var ErrOERInvalidLicense = errors.New("invalid or missing OER license")

type OERService interface {
	// ImportPack validates every row's license against the allowlist and then
	// upserts the pack. Existing activity IDs are skipped so learner progress
	// is never orphaned. The report is honest: imported / skipped / rejected.
	ImportPack(ctx context.Context, pack domain.OERPack) (domain.OERImportReport, error)
}