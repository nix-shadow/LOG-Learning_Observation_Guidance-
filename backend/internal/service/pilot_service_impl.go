package service

import (
	"context"
	"time"

	"log-backend/internal/domain"
)

type pilotService struct {
	repo domain.PilotRepository
	// posterExists checks the activity actually exists before recording a
	// scan — injected so tests can stub it, production uses the activity repo.
	posterExists func(ctx context.Context, activityID string) (bool, error)
}

func NewPilotService(repo domain.PilotRepository, posterExists func(ctx context.Context, activityID string) (bool, error)) PilotService {
	return &pilotService{repo: repo, posterExists: posterExists}
}

func (s *pilotService) RecordScan(ctx context.Context, posterID, source string) (*domain.PilotScan, error) {
	if source == "" {
		source = "qr"
	}
	exists, err := s.posterExists(ctx, posterID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrPilotPosterNotFound
	}
	scan := &domain.PilotScan{PosterID: posterID, Source: source, CreatedAt: time.Now()}
	if err := s.repo.CreateScan(ctx, scan); err != nil {
		return nil, err
	}
	return scan, nil
}

func (s *pilotService) MarkStarted(ctx context.Context, scanID uint) error {
	return s.repo.MarkStarted(ctx, scanID)
}

func (s *pilotService) Stats(ctx context.Context) (domain.PilotStats, error) {
	return s.repo.Stats(ctx)
}