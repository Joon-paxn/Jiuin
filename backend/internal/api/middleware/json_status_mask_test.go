package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
)

func TestMaskSuccessfulJSONMasksOnlyConfiguredSuccessfulMusicRoutes(t *testing.T) {
	t.Parallel()

	handler := MaskSuccessfulJSON(JSONStatusMaskConfig{
		Enabled: true,
		Status:  http.StatusTeapot,
		Routes:  []string{"/api/v1/music", "/api/v1/music/{id}"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]string{"path": r.URL.Path})
	}))

	for _, path := range []string{"/api/v1/music", "/api/v1/music/track-1"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusTeapot {
			t.Fatalf("%s status = %d, want %d", path, recorder.Code, http.StatusTeapot)
		}
		if recorder.Header().Get(MaskedJSONHeader) != "1" {
			t.Fatalf("%s %s header = %q, want 1", path, MaskedJSONHeader, recorder.Header().Get(MaskedJSONHeader))
		}
		if recorder.Body.String() == "" || recorder.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Fatalf("%s did not preserve JSON success response", path)
		}
	}
}

func TestMaskSuccessfulJSONNeverMasksErrorsMediaOrHead(t *testing.T) {
	t.Parallel()

	handler := MaskSuccessfulJSON(JSONStatusMaskConfig{
		Enabled: true,
		Status:  http.StatusTeapot,
		Routes:  []string{"/api/v1/music", "/api/v1/music/{id}"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/music/missing" {
			response.Error(w, http.StatusNotFound, "music track was not found")
			return
		}
		response.Success(w, map[string]string{"path": r.URL.Path})
	}))

	tests := []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/v1/music/missing", http.StatusNotFound},
		{http.MethodGet, "/media/music/full/track.mp3", http.StatusOK},
		{http.MethodHead, "/api/v1/music", http.StatusOK},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		if recorder.Code != test.status {
			t.Fatalf("%s %s status = %d, want %d", test.method, test.path, recorder.Code, test.status)
		}
		if recorder.Header().Get(MaskedJSONHeader) != "" {
			t.Fatalf("%s %s was unexpectedly marked", test.method, test.path)
		}
	}
}
