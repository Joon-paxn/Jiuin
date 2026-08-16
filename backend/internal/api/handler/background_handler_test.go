package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBackgroundHandlerReturnsOneAllowlistedURL(t *testing.T) {
	t.Parallel()

	handler := NewBackgroundHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/background/random", nil)
	recorder := httptest.NewRecorder()
	handler.Random(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != http.StatusOK || body.Data.URL == "" {
		t.Fatalf("unexpected response: %#v", body)
	}

	for _, allowed := range backgroundURLAllowlist {
		if body.Data.URL == allowed {
			return
		}
	}
	t.Fatalf("URL %q is not allowlisted", body.Data.URL)
}
