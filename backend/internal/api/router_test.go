package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

type stubSiteService struct{}

type stubMusicService struct{}
type stubStatisticsService struct{}
type stubStatusService struct{}
type stubLinkService struct{}
type stubResourceService struct{}

func (stubSiteService) Info(context.Context) (model.SiteInfo, error) {
	return model.SiteInfo{Name: "霁雪居", Project: "Jiuin", Domain: "Jiuin.cn"}, nil
}

func (stubSiteService) Copyright(context.Context) model.Copyright {
	return model.Copyright{Year: 2026, Text: "Jiuin.cn"}
}

func (stubSiteService) Shared(context.Context) (model.SharedSiteConfiguration, error) {
	return model.SharedSiteConfiguration{
		Site:      model.SiteInfo{Name: "霁雪居", Project: "Jiuin", Domain: "Jiuin.cn"},
		Copyright: model.Copyright{Year: 2026, Text: "Jiuin.cn"},
	}, nil
}

func (stubMusicService) List(context.Context) ([]model.MusicTrack, error) {
	return []model.MusicTrack{}, nil
}

func (stubStatisticsService) Record(_ context.Context, path string) (model.PageStatistics, error) {
	return model.PageStatistics{Path: path, Views: 1, LastVisitedAt: time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)}, nil
}

func (stubStatisticsService) Summary(context.Context) (model.SiteStatistics, error) {
	return model.SiteStatistics{TotalViews: 1, Pages: []model.PageStatistics{}}, nil
}

func (stubStatusService) Get(context.Context) (model.EcosystemStatus, error) {
	return model.EcosystemStatus{Site: "online", API: "online", Services: []model.ServiceStatus{}}, nil
}

func (stubLinkService) List(context.Context) ([]model.ExternalLink, error) {
	return []model.ExternalLink{}, nil
}

func (stubResourceService) List(context.Context) ([]model.ResourceDescriptor, error) {
	return []model.ResourceDescriptor{}, nil
}

func TestRouterExposesVersionedPublicEndpoints(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewRouter(
		stubSiteService{},
		stubMusicService{},
		stubStatisticsService{},
		stubStatusService{},
		stubLinkService{},
		stubResourceService{},
		"test-shared-service-token",
		[]string{"http://localhost:5173"},
		logger,
	)

	testCases := []struct {
		name string
		path string
	}{
		{name: "health", path: "/api/v1/health"},
		{name: "site info", path: "/api/v1/site/info"},
		{name: "copyright", path: "/api/v1/site/copyright"},
		{name: "shared site configuration", path: "/api/v1/site"},
		{name: "music list", path: "/api/v1/music/list"},
		{name: "statistics", path: "/api/v1/statistics"},
		{name: "status", path: "/api/v1/status"},
		{name: "links", path: "/api/v1/links"},
		{name: "resources", path: "/api/v1/resources"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			request.Header.Set("Origin", "http://localhost:5173")
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
			if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
				t.Fatal("expected configured CORS origin header")
			}

			var body struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}
			if body.Code != http.StatusOK || body.Message != "success" || len(body.Data) == 0 {
				t.Fatalf("unexpected response envelope: %#v", body)
			}
		})
	}
}

func TestRouterProtectsStatisticsWrites(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewRouter(
		stubSiteService{}, stubMusicService{}, stubStatisticsService{}, stubStatusService{}, stubLinkService{}, stubResourceService{},
		"test-shared-service-token", []string{"http://localhost:5173"}, logger,
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/statistics/visit", strings.NewReader(`{"path":"/"}`))
	request.Header.Set("Authorization", "Bearer test-shared-service-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	unauthorizedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/statistics/visit", strings.NewReader(`{"path":"/"}`))
	unauthorizedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedRecorder, unauthorizedRequest)
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorizedRecorder.Code, http.StatusUnauthorized)
	}
}
