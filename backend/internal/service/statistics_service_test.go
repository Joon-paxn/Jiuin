package service

import (
	"context"
	"testing"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

func TestStatisticsServiceRecordsRelativePagePaths(t *testing.T) {
	t.Parallel()

	service := statisticsService{
		repository: repository.NewMemoryStatisticsRepository(),
		now: func() time.Time {
			return time.Date(2026, time.August, 10, 10, 0, 0, 0, time.UTC)
		},
	}

	first, err := service.Record(context.Background(), "/?from=home")
	if err != nil {
		t.Fatalf("Record() returned an error: %v", err)
	}
	if first.Path != "/" || first.Views != 1 {
		t.Fatalf("first record = %#v, want normalized root path with one view", first)
	}

	if _, err := service.Record(context.Background(), "https://outside.example"); err == nil {
		t.Fatal("expected an absolute URL to be rejected")
	}

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() returned an error: %v", err)
	}
	if summary.TotalViews != 1 || len(summary.Pages) != 1 {
		t.Fatalf("summary = %#v, want one total and one page", summary)
	}
}
