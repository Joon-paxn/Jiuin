package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type streamMusicService struct {
	asset model.MusicAsset
	err   error
}

type deadlineResponseRecorder struct {
	*httptest.ResponseRecorder
	deadline time.Time
}

func (recorder *deadlineResponseRecorder) SetWriteDeadline(deadline time.Time) error {
	recorder.deadline = deadline
	return nil
}

func (stub streamMusicService) List(context.Context) ([]model.MusicTrack, error) {
	return nil, nil
}

func (stub streamMusicService) Open(context.Context, string) (model.MusicAsset, error) {
	return stub.asset, stub.err
}

func TestMusicHandlerStreamsByteRanges(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "Test Artist - Test Song.mp3")
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatalf("write music fixture: %v", err)
	}
	handler := NewMusicHandler(streamMusicService{asset: model.MusicAsset{Path: path, Name: filepath.Base(path)}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/test-track", nil)
	request.Header.Set("Range", "bytes=2-5")
	request.SetPathValue("id", "test-track")
	recorder := httptest.NewRecorder()

	handler.Stream(recorder, request)

	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusPartialContent)
	}
	if recorder.Body.String() != "2345" {
		t.Fatalf("body = %q, want 2345", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("content type = %q, want audio/mpeg", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Cache-Control") == "" {
		t.Fatal("expected cache header")
	}
}

func TestMusicHandlerStreamSupportsResponseDeadlines(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "Test Artist - Test Song.mp3")
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatalf("write music fixture: %v", err)
	}
	handler := NewMusicHandler(streamMusicService{asset: model.MusicAsset{Path: path, Name: filepath.Base(path)}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/test-track", nil)
	request.SetPathValue("id", "test-track")
	recorder := &deadlineResponseRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.Stream(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !recorder.deadline.IsZero() {
		t.Fatalf("write deadline = %v, want zero deadline for stream", recorder.deadline)
	}
}

func TestMusicHandlerRejectsUnknownTrack(t *testing.T) {
	t.Parallel()

	handler := NewMusicHandler(streamMusicService{err: service.ErrMusicNotFound}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/unknown", nil)
	request.SetPathValue("id", "unknown")
	recorder := httptest.NewRecorder()

	handler.Stream(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestMusicHandlerPreservesUnexpectedOpenErrors(t *testing.T) {
	t.Parallel()

	handler := NewMusicHandler(streamMusicService{err: errors.New("disk offline")}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/broken", nil)
	request.SetPathValue("id", "broken")
	recorder := httptest.NewRecorder()

	handler.Stream(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestPublicMusicURLOnlyTrustsForwardedProtocolFromLoopbackProxy(t *testing.T) {
	t.Parallel()

	directRequest := httptest.NewRequest(http.MethodGet, "http://jiuin.cn/api/v1/music/list", nil)
	directRequest.RemoteAddr = "198.51.100.12:49152"
	directRequest.Header.Set("X-Forwarded-Proto", "https")
	if url := publicMusicURL(directRequest, "/media/music/track"); url != "http://jiuin.cn/media/music/track" {
		t.Fatalf("direct client URL = %q, want untrusted forwarded protocol to be ignored", url)
	}

	proxiedRequest := httptest.NewRequest(http.MethodGet, "http://jiuin.cn/api/v1/music/list", nil)
	proxiedRequest.RemoteAddr = "127.0.0.1:49152"
	proxiedRequest.Header.Set("X-Forwarded-Proto", "https")
	if url := publicMusicURL(proxiedRequest, "/media/music/track"); url != "https://jiuin.cn/media/music/track" {
		t.Fatalf("proxied URL = %q, want trusted HTTPS URL", url)
	}
}
