package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

type StatisticsService interface {
	Record(context.Context, string) (model.PageStatistics, error)
	Summary(context.Context) (model.SiteStatistics, error)
}

type statisticsService struct {
	repository repository.StatisticsRepository
	now        func() time.Time
}

func NewStatisticsService(repository repository.StatisticsRepository) StatisticsService {
	return statisticsService{repository: repository, now: time.Now}
}

func (service statisticsService) Record(ctx context.Context, path string) (model.PageStatistics, error) {
	path, err := validateVisitPath(path)
	if err != nil {
		return model.PageStatistics{}, err
	}

	return service.repository.Record(ctx, model.VisitRecord{Path: path, VisitedAt: service.now()})
}

func (service statisticsService) Summary(ctx context.Context) (model.SiteStatistics, error) {
	return service.repository.Summary(ctx)
}

func validateVisitPath(value string) (string, error) {
	if len(value) == 0 || len(value) > 512 || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "", fmt.Errorf("path must be a relative site path")
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || strings.ContainsRune(parsed.Path, '\x00') {
		return "", fmt.Errorf("path must be a valid relative site path")
	}

	return parsed.Path, nil
}
