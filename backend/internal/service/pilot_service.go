package service

import (
	"context"
	"errors"

	"log-backend/internal/domain"
)

var (
	// ErrPilotPosterNotFound is returned when a scan names a poster whose
	// activity does not exist — the pilot never records scans against
	// fabricated posters.
	ErrPilotPosterNotFound = errors.New("poster activity not found")
	// ErrPilotScanNotFound is returned when marking a start for an unknown
	// scan id (e.g. the scan was recorded on another device).
	ErrPilotScanNotFound = errors.New("scan not found")
)

type PilotService interface {
	// RecordScan persists one poster scan (WP-3.3 RC-10). No IP or device
	// data is stored — privacy by design.
	RecordScan(ctx context.Context, posterID, source string) (*domain.PilotScan, error)
	// MarkStarted flips a scan to started when the learner clicks through.
	MarkStarted(ctx context.Context, scanID uint) error
	// Stats returns honest aggregates over real scan rows.
	Stats(ctx context.Context) (domain.PilotStats, error)
}