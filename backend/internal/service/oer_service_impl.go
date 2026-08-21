package service

import (
	"context"
	"fmt"
	"strings"

	"log-backend/internal/domain"
)

type oerService struct {
	activityRepo domain.ActivityRepository
}

func NewOERService(activityRepo domain.ActivityRepository) OERService {
	return &oerService{activityRepo: activityRepo}
}

// ImportPack validates every row's license before touching the database, then
// imports via CreateMany (skipping existing IDs). A row with an empty or
// unknown license is rejected with its index + id + reason; the rest of the
// pack still imports, and the report tells the caller exactly what happened.
func (s *oerService) ImportPack(ctx context.Context, pack domain.OERPack) (domain.OERImportReport, error) {
	var report domain.OERImportReport
	var toImport []domain.Activity

	for i, a := range pack.Activities {
		rowNo := i + 1
		if a.ID == "" {
			report.Errors = append(report.Errors, domain.OERImportRowError{
				Row: rowNo, Reason: "missing id — an import row must carry its own activity id",
			})
			continue
		}
		if !domain.IsAllowedOERLicense(a.License) {
			reason := fmt.Sprintf("license %q is not in the OER allowlist", a.License)
			if a.License == "" {
				reason = "missing license — every imported activity must declare its OER license"
			}
			report.Errors = append(report.Errors, domain.OERImportRowError{
				Row: rowNo, ID: a.ID, Reason: reason,
			})
			continue
		}
		if a.License != "Own work (LOG team)" && strings.TrimSpace(a.Attribution) == "" {
			report.Errors = append(report.Errors, domain.OERImportRowError{
				Row: rowNo, ID: a.ID,
				Reason: "attribution required — third-party OER must name its author",
			})
			continue
		}
		// Normalize the license URL so the UI always renders a real deed link.
		if a.LicenseURL == "" && domain.OERLicenseURLs[a.License] != "" {
			a.LicenseURL = domain.OERLicenseURLs[a.License]
		}
		toImport = append(toImport, a)
	}

	imported, skipped, err := s.activityRepo.CreateMany(ctx, toImport)
	if err != nil {
		return report, err
	}
	report.Imported = imported
	report.Skipped = skipped
	return report, nil
}