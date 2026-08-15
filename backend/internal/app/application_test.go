package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/config"
)

func TestApplicationReachesReadyAndShutsDown(t *testing.T) {
	t.Parallel()
	application, err := New(config.Config{
		Environment: "development",
		Server:      config.ServerConfig{Host: "127.0.0.1", Port: "0", ReadHeaderTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second, IdleTimeout: time.Second, ShutdownTimeout: time.Second},
		Site:        config.SiteConfig{Name: "Jiuin", Project: "Jiuin", Domain: "jiuin.cn"},
		Music:       config.MusicConfig{Directory: t.TempDir(), MaxUploadSize: 1024, FFmpegPath: "ffmpeg", FFprobePath: "ffprobe", FullBitrate: "320k", LiteBitrate: "128k", OutputCodec: "libmp3lame", WorkerCount: 1, ProcessingTimeout: time.Minute, AdminToken: "test-music-admin-token-with-at-least-32-characters"},
		CORS:        config.CORSConfig{AllowedOrigins: []string{"http://localhost:5173"}},
		Ecosystem:   config.EcosystemConfig{SharedServiceToken: "test-service-token-with-at-least-32-characters", MainSiteStatus: "online", BlogStatus: "unknown"},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runResult := make(chan error, 1)
	go func() { runResult <- application.Run(context.Background()) }()
	deadline := time.Now().Add(2 * time.Second)
	for !application.Ready() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !application.Ready() {
		t.Fatal("application did not become ready")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := application.Shutdown(shutdownContext); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := <-runResult; err != nil {
		t.Fatalf("Run: %v", err)
	}
	if application.Ready() {
		t.Fatal("application remained ready after shutdown")
	}
}
