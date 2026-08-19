package service

import (
	"context"
	"testing"
	"time"

	"log-backend/internal/domain"
)

// --- AttemptStats completion-time semantics (WP-0.2 research round) ---

func TestAttemptStatsCompletedAtHonestClamp(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		ms   int64
		want time.Time
	}{
		{
			name: "no timestamp falls back to now",
			ms:   0,
			want: now,
		},
		{
			name: "recent offline completion keeps its real time",
			ms:   now.Add(-48 * time.Hour).UnixMilli(),
			want: now.Add(-48 * time.Hour),
		},
		{
			name: "ancient timestamp clamps to 14 days",
			ms:   now.Add(-30 * 24 * time.Hour).UnixMilli(),
			want: now.Add(-14 * 24 * time.Hour),
		},
		{
			name: "future clock skew clamps to +24h",
			ms:   now.Add(48 * time.Hour).UnixMilli(),
			want: now.Add(24 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := domain.AttemptStats{CompletedAtUnixMs: tt.ms}
			got := stats.CompletedAt(now)
			if !got.Equal(tt.want) {
				t.Fatalf("CompletedAt = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttemptStatsLocationFallsBackToUTC(t *testing.T) {
	if got := (domain.AttemptStats{}).Location(); got != time.UTC {
		t.Fatalf("empty timezone must resolve to UTC, got %v", got)
	}
	if got := (domain.AttemptStats{TimezoneIANA: "Not/AZone"}).Location(); got != time.UTC {
		t.Fatalf("invalid timezone must resolve to UTC, got %v", got)
	}
	if got := (domain.AttemptStats{TimezoneIANA: "Asia/Kathmandu"}).Location(); got.String() != "Asia/Kathmandu" {
		t.Fatalf("valid timezone must resolve to Asia/Kathmandu, got %v", got)
	}
}

// --- GetChartData honesty (AGENTS.md §1: never fabricate) ---

type fakeLearnerDataRepo struct {
	activities []domain.DailyActivity
}

func (f *fakeLearnerDataRepo) FindObservations(ctx context.Context, learnerID string) ([]domain.Observation, error) {
	return nil, nil
}

func (f *fakeLearnerDataRepo) FindGuidance(ctx context.Context, learnerID string) ([]domain.Guidance, error) {
	return nil, nil
}

func (f *fakeLearnerDataRepo) FindDailyActivities(ctx context.Context, learnerID string) ([]domain.DailyActivity, error) {
	return f.activities, nil
}

func TestGetChartDataHonestEmpty(t *testing.T) {
	svc := NewLearnerService(nil, nil, nil, &fakeLearnerDataRepo{activities: nil}, nil)
	data, err := svc.GetChartData(context.Background(), "user-empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("a learner with no activity must get an EMPTY list (honest state), got %d fabricated rows", len(data))
	}
}

func TestGetChartDataOnlyRealRows(t *testing.T) {
	real := domain.DailyActivity{
		ID:        "dly-1",
		LearnerID: "user-1",
		Date:      time.Now(),
		DayName:   "Wed",
		Score:     100.0,
		Duration:  420,
	}
	svc := NewLearnerService(nil, nil, nil, &fakeLearnerDataRepo{activities: []domain.DailyActivity{real}}, nil)
	data, err := svc.GetChartData(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected exactly 1 real row, got %d", len(data))
	}
	if data[0]["score"] != 100.0 || data[0]["duration"] != 420 {
		t.Fatalf("chart rows must carry real observed values, got %+v", data[0])
	}
}
