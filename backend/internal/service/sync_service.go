package service

import (
	"context"
	"log-backend/internal/domain"
)

type SyncService interface {
	ProcessBulkSync(ctx context.Context, learnerID string, data []domain.SyncRequestItem) (int, error)
}

type syncService struct {
	syncRepo domain.SyncRepository
}

func NewSyncService(syncRepo domain.SyncRepository) SyncService {
	return &syncService{syncRepo: syncRepo}
}

func (s *syncService) ProcessBulkSync(ctx context.Context, learnerID string, data []domain.SyncRequestItem) (int, error) {
	return s.syncRepo.SyncBulk(ctx, learnerID, data)
}
