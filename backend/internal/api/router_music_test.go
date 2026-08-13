package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

func TestRouterListsAndStreamsLocalMusic(t *testing.T) {
	directory := t.TempDir()
	payload := []byte("abcdef")
	if err := os.WriteFile(filepath.Join(directory, "Artist - Track.mp3"), payload, 0o644); err != nil {
		t.Fatalf("write music fixture: %v", err)
	}

	musicRepository, err := repository.NewFilesystemMusicRepository(directory)
	if err != nil {
		t.Fatalf("NewFilesystemMusicRepository() error = %v", err)
	}
	handler := newMusicTestRouter(service.NewMusicService(musicRepository))

	listRequest := httptest.NewRequest(http.MethodGet, "http://attacker.example/api/v1/music/list", nil)
	listRequest.RemoteAddr = "198.51.100.12:49152"
	listRequest.Header.Set("Origin", "http://app.test")
	listRecorder := httptest.NewRecorder()
	handler.ServeHTTP(listRecorder, listRequest)

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRecorder.Code, http.StatusOK)
	}
	if listRecorder.Header().Get("Access-Control-Allow-Origin") != "http://app.test" {
		t.Fatalf("list CORS origin = %q, want configured origin", listRecorder.Header().Get("Access-Control-Allow-Origin"))
	}

	var listBody struct {
		Code int                `json:"code"`
		Data []model.MusicTrack `json:"data"`
	}
	if err := json.NewDecoder(listRecorder.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode music list: %v", err)
	}
	if listBody.Code != http.StatusOK || len(listBody.Data) != 1 {
		t.Fatalf("music list = %#v, want one successful track", listBody)
	}
	track := listBody.Data[0]
	wantSourceURL := "/media/music/" + track.ID
	if track.SourceURL != wantSourceURL {
		t.Fatalf("sourceUrl = %q, want %q", track.SourceURL, wantSourceURL)
	}

	streamRequest := httptest.NewRequest(http.MethodGet, track.SourceURL, nil)
	streamRequest.Header.Set("Origin", "http://app.test")
	streamRequest.Header.Set("Range", "bytes=1-3")
	streamRecorder := httptest.NewRecorder()
	handler.ServeHTTP(streamRecorder, streamRequest)

	if streamRecorder.Code != http.StatusPartialContent {
		t.Fatalf("range status = %d, want %d", streamRecorder.Code, http.StatusPartialContent)
	}
	if streamRecorder.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("Content-Type = %q, want audio/mpeg", streamRecorder.Header().Get("Content-Type"))
	}
	if streamRecorder.Header().Get("Content-Range") != "bytes 1-3/6" {
		t.Fatalf("Content-Range = %q, want bytes 1-3/6", streamRecorder.Header().Get("Content-Range"))
	}
	if streamRecorder.Header().Get("Cache-Control") != "public, max-age=3600, must-revalidate" {
		t.Fatalf("Cache-Control = %q", streamRecorder.Header().Get("Cache-Control"))
	}
	if streamRecorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", streamRecorder.Header().Get("X-Content-Type-Options"))
	}
	if !bytes.Equal(streamRecorder.Body.Bytes(), []byte("bcd")) {
		t.Fatalf("range body = %q, want bcd", streamRecorder.Body.Bytes())
	}

	etag := streamRecorder.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag on music response")
	}
	conditionalRequest := httptest.NewRequest(http.MethodGet, track.SourceURL, nil)
	conditionalRequest.Header.Set("If-None-Match", etag)
	conditionalRecorder := httptest.NewRecorder()
	handler.ServeHTTP(conditionalRecorder, conditionalRequest)
	if conditionalRecorder.Code != http.StatusNotModified {
		t.Fatalf("conditional status = %d, want %d", conditionalRecorder.Code, http.StatusNotModified)
	}

	headRequest := httptest.NewRequest(http.MethodHead, track.SourceURL, nil)
	headRecorder := httptest.NewRecorder()
	handler.ServeHTTP(headRecorder, headRequest)
	if headRecorder.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want %d", headRecorder.Code, http.StatusOK)
	}
	if headRecorder.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want 0", headRecorder.Body.Len())
	}

	invalidRangeRequest := httptest.NewRequest(http.MethodGet, track.SourceURL, nil)
	invalidRangeRequest.Header.Set("Range", "bytes=99-100")
	invalidRangeRecorder := httptest.NewRecorder()
	handler.ServeHTTP(invalidRangeRecorder, invalidRangeRequest)
	if invalidRangeRecorder.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("invalid range status = %d, want %d", invalidRangeRecorder.Code, http.StatusRequestedRangeNotSatisfiable)
	}

	missingRequest := httptest.NewRequest(http.MethodGet, "http://music.test/media/music/not-a-track", nil)
	missingRecorder := httptest.NewRecorder()
	handler.ServeHTTP(missingRecorder, missingRequest)
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("traversal status = %d, want %d", missingRecorder.Code, http.StatusNotFound)
	}
}

func newMusicTestRouter(musicService service.MusicService) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(
		stubSiteService{},
		musicService,
		stubStatisticsService{},
		stubStatusService{},
		stubLinkService{},
		stubResourceService{},
		"test-shared-service-token",
		"test-music-admin-token",
		[]string{"http://app.test"},
		logger,
	)
}
