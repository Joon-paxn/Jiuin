package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

type stubSiteService struct{}

func (stubSiteService) Info(context.Context) (model.SiteInfo, error) {
	return model.SiteInfo{Name: "霁雪居", Project: "Jiuin", Domain: "Jiuin.cn"}, nil
}

func (stubSiteService) Copyright(context.Context) model.Copyright {
	return model.Copyright{Year: 2026, Text: "Jiuin.cn"}
}

func TestRouterExposesVersionedPublicEndpoints(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewRouter(stubSiteService{}, []string{"http://localhost:5173"}, logger)

	testCases := []struct {
		name string
		path string
	}{
		{name: "health", path: "/api/v1/health"},
		{name: "site info", path: "/api/v1/site/info"},
		{name: "copyright", path: "/api/v1/site/copyright"},
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
