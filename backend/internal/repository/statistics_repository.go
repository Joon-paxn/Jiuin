package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

// StatisticsRepository is intentionally separate from content repositories.
// A database-backed implementation can replace this in-memory implementation later.
type StatisticsRepository interface {
	Record(context.Context, model.VisitRecord) (model.PageStatistics, error)
	Summary(context.Context) (model.SiteStatistics, error)
}

type memoryStatisticsRepository struct {
	mu    sync.RWMutex
	pages map[string]model.PageStatistics
	total int64
}

func NewMemoryStatisticsRepository() StatisticsRepository {
	return &memoryStatisticsRepository{pages: make(map[string]model.PageStatistics)}
}

func (repository *memoryStatisticsRepository) Record(_ context.Context, visit model.VisitRecord) (model.PageStatistics, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	page := repository.pages[visit.Path]
	page.Path = visit.Path
	page.Views++
	page.LastVisitedAt = visit.VisitedAt.UTC()
	repository.pages[visit.Path] = page
	repository.total++

	return page, nil
}

func (repository *memoryStatisticsRepository) Summary(context.Context) (model.SiteStatistics, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	pages := make([]model.PageStatistics, 0, len(repository.pages))
	for _, page := range repository.pages {
		pages = append(pages, page)
	}
	sort.Slice(pages, func(left, right int) bool {
		return pages[left].Path < pages[right].Path
	})

	return model.SiteStatistics{TotalViews: repository.total, Pages: pages}, nil
}
