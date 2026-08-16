package config

import "testing"

func TestLoadReadsRequiredEnvironmentConfiguration(t *testing.T) {
	t.Setenv("JIUIN_ENV", "development")
	t.Setenv("JIUIN_SERVER_HOST", "127.0.0.1")
	t.Setenv("JIUIN_SERVER_PORT", "8080")
	t.Setenv("JIUIN_SERVER_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("JIUIN_SERVER_READ_TIMEOUT", "15s")
	t.Setenv("JIUIN_SERVER_WRITE_TIMEOUT", "30s")
	t.Setenv("JIUIN_SERVER_IDLE_TIMEOUT", "60s")
	t.Setenv("JIUIN_SERVER_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("JIUIN_SITE_NAME", "霁雪居")
	t.Setenv("JIUIN_SITE_PROJECT", "Jiuin")
	t.Setenv("JIUIN_SITE_DOMAIN", "Jiuin.cn")
	t.Setenv("JIUIN_MUSIC_DIRECTORY", "storage/music")
	t.Setenv("JIUIN_MUSIC_MAX_UPLOAD_SIZE", "100MB")
	t.Setenv("JIUIN_FFMPEG_PATH", "ffmpeg")
	t.Setenv("JIUIN_FFPROBE_PATH", "ffprobe")
	t.Setenv("JIUIN_MUSIC_FULL_BITRATE", "320k")
	t.Setenv("JIUIN_MUSIC_LITE_BITRATE", "128k")
	t.Setenv("JIUIN_MUSIC_OUTPUT_CODEC", "libmp3lame")
	t.Setenv("JIUIN_MUSIC_WORKER_COUNT", "2")
	t.Setenv("JIUIN_MUSIC_ADMIN_TOKEN", "test-music-admin-token-with-at-least-32-characters")
	t.Setenv("JIUIN_CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://jiuin.cn")
	t.Setenv("JIUIN_SHARED_SERVICE_TOKEN", "test-shared-service-token-with-at-least-32-characters")
	t.Setenv("JIUIN_MAIN_SITE_STATUS", "online")
	t.Setenv("JIUIN_BLOG_STATUS", "unknown")
	t.Setenv("JIUIN_EXTERNAL_LINKS_JSON", "[]")
	t.Setenv("JIUIN_RESOURCE_MANIFEST_JSON", "[]")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if config.Server.Address() != "127.0.0.1:8080" {
		t.Fatalf("Server.Address() = %q, want 127.0.0.1:8080", config.Server.Address())
	}
	if len(config.CORS.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins = %#v, want two configured origins", config.CORS.AllowedOrigins)
	}
	if config.Music.Directory != "storage/music" {
		t.Fatalf("Music.Directory = %q, want storage/music", config.Music.Directory)
	}
	if config.HTTPMask.Enabled || config.HTTPMask.Status != 418 {
		t.Fatalf("HTTPMask = %#v, want disabled 418 masking by default", config.HTTPMask)
	}
}

func TestHTTPMaskConfigurationValidation(t *testing.T) {
	if enabled, err := parseOptionalBoolean("JIUIN_HTTP_MASK_ENABLED", "true"); err != nil || !enabled {
		t.Fatalf("parseOptionalBoolean(true) = (%t, %v), want (true, nil)", enabled, err)
	}
	if _, err := parseOptionalBoolean("JIUIN_HTTP_MASK_ENABLED", "not-a-boolean"); err == nil {
		t.Fatal("parseOptionalBoolean accepted invalid value")
	}
	if status, err := parseOptionalHTTPMaskStatus("418"); err != nil || status != 418 {
		t.Fatalf("parseOptionalHTTPMaskStatus(418) = (%d, %v), want (418, nil)", status, err)
	}
	if _, err := parseOptionalHTTPMaskStatus("200"); err == nil {
		t.Fatal("parseOptionalHTTPMaskStatus accepted non-418 status")
	}
}

func TestLoadRejectsInsecureProductionOrigin(t *testing.T) {
	t.Setenv("JIUIN_ENV", "production")
	t.Setenv("JIUIN_SERVER_HOST", "127.0.0.1")
	t.Setenv("JIUIN_SERVER_PORT", "8080")
	t.Setenv("JIUIN_SERVER_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("JIUIN_SERVER_READ_TIMEOUT", "15s")
	t.Setenv("JIUIN_SERVER_WRITE_TIMEOUT", "30s")
	t.Setenv("JIUIN_SERVER_IDLE_TIMEOUT", "60s")
	t.Setenv("JIUIN_SERVER_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("JIUIN_SITE_NAME", "Jiuin")
	t.Setenv("JIUIN_SITE_PROJECT", "Jiuin")
	t.Setenv("JIUIN_SITE_DOMAIN", "jiuin.cn")
	t.Setenv("JIUIN_MUSIC_DIRECTORY", "storage/music")
	t.Setenv("JIUIN_MUSIC_MAX_UPLOAD_SIZE", "100MB")
	t.Setenv("JIUIN_FFMPEG_PATH", "ffmpeg")
	t.Setenv("JIUIN_FFPROBE_PATH", "ffprobe")
	t.Setenv("JIUIN_MUSIC_FULL_BITRATE", "320k")
	t.Setenv("JIUIN_MUSIC_LITE_BITRATE", "128k")
	t.Setenv("JIUIN_MUSIC_OUTPUT_CODEC", "libmp3lame")
	t.Setenv("JIUIN_MUSIC_WORKER_COUNT", "2")
	t.Setenv("JIUIN_MUSIC_ADMIN_TOKEN", "production-music-admin-token-with-at-least-32-characters")
	t.Setenv("JIUIN_CORS_ALLOWED_ORIGINS", "http://jiuin.cn")
	t.Setenv("JIUIN_SHARED_SERVICE_TOKEN", "production-token-with-at-least-32-characters")
	t.Setenv("JIUIN_MAIN_SITE_STATUS", "online")
	t.Setenv("JIUIN_BLOG_STATUS", "unknown")
	t.Setenv("JIUIN_EXTERNAL_LINKS_JSON", "[]")
	t.Setenv("JIUIN_RESOURCE_MANIFEST_JSON", "[]")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded with an insecure production CORS origin")
	}
}

func TestLoadRejectsUnknownEnvironment(t *testing.T) {
	t.Setenv("JIUIN_ENV", "test")
	t.Setenv("JIUIN_SERVER_HOST", "127.0.0.1")
	t.Setenv("JIUIN_SERVER_PORT", "8080")
	t.Setenv("JIUIN_SERVER_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("JIUIN_SERVER_READ_TIMEOUT", "15s")
	t.Setenv("JIUIN_SERVER_WRITE_TIMEOUT", "30s")
	t.Setenv("JIUIN_SERVER_IDLE_TIMEOUT", "60s")
	t.Setenv("JIUIN_SERVER_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("JIUIN_SITE_NAME", "Jiuin")
	t.Setenv("JIUIN_SITE_PROJECT", "Jiuin")
	t.Setenv("JIUIN_SITE_DOMAIN", "jiuin.cn")
	t.Setenv("JIUIN_MUSIC_DIRECTORY", "storage/music")
	t.Setenv("JIUIN_MUSIC_MAX_UPLOAD_SIZE", "100MB")
	t.Setenv("JIUIN_FFMPEG_PATH", "ffmpeg")
	t.Setenv("JIUIN_FFPROBE_PATH", "ffprobe")
	t.Setenv("JIUIN_MUSIC_FULL_BITRATE", "320k")
	t.Setenv("JIUIN_MUSIC_LITE_BITRATE", "128k")
	t.Setenv("JIUIN_MUSIC_OUTPUT_CODEC", "libmp3lame")
	t.Setenv("JIUIN_MUSIC_WORKER_COUNT", "2")
	t.Setenv("JIUIN_MUSIC_ADMIN_TOKEN", "development-music-admin-token-with-at-least-32-characters")
	t.Setenv("JIUIN_CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	t.Setenv("JIUIN_SHARED_SERVICE_TOKEN", "development-token-with-at-least-32-characters")
	t.Setenv("JIUIN_MAIN_SITE_STATUS", "online")
	t.Setenv("JIUIN_BLOG_STATUS", "unknown")
	t.Setenv("JIUIN_EXTERNAL_LINKS_JSON", "[]")
	t.Setenv("JIUIN_RESOURCE_MANIFEST_JSON", "[]")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded with an unknown environment")
	}
}

func TestLoadRejectsProductionPlaceholderServiceToken(t *testing.T) {
	t.Setenv("JIUIN_ENV", "production")
	t.Setenv("JIUIN_SERVER_HOST", "127.0.0.1")
	t.Setenv("JIUIN_SERVER_PORT", "8080")
	t.Setenv("JIUIN_SERVER_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("JIUIN_SERVER_READ_TIMEOUT", "15s")
	t.Setenv("JIUIN_SERVER_WRITE_TIMEOUT", "30s")
	t.Setenv("JIUIN_SERVER_IDLE_TIMEOUT", "60s")
	t.Setenv("JIUIN_SERVER_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("JIUIN_SITE_NAME", "Jiuin")
	t.Setenv("JIUIN_SITE_PROJECT", "Jiuin")
	t.Setenv("JIUIN_SITE_DOMAIN", "jiuin.cn")
	t.Setenv("JIUIN_MUSIC_DIRECTORY", "storage/music")
	t.Setenv("JIUIN_MUSIC_MAX_UPLOAD_SIZE", "100MB")
	t.Setenv("JIUIN_FFMPEG_PATH", "ffmpeg")
	t.Setenv("JIUIN_FFPROBE_PATH", "ffprobe")
	t.Setenv("JIUIN_MUSIC_FULL_BITRATE", "320k")
	t.Setenv("JIUIN_MUSIC_LITE_BITRATE", "128k")
	t.Setenv("JIUIN_MUSIC_OUTPUT_CODEC", "libmp3lame")
	t.Setenv("JIUIN_MUSIC_WORKER_COUNT", "2")
	t.Setenv("JIUIN_MUSIC_ADMIN_TOKEN", "production-music-admin-token-with-at-least-32-characters")
	t.Setenv("JIUIN_CORS_ALLOWED_ORIGINS", "https://jiuin.cn")
	t.Setenv("JIUIN_SHARED_SERVICE_TOKEN", "replace-with-a-long-random-production-token")
	t.Setenv("JIUIN_MAIN_SITE_STATUS", "online")
	t.Setenv("JIUIN_BLOG_STATUS", "unknown")
	t.Setenv("JIUIN_EXTERNAL_LINKS_JSON", "[]")
	t.Setenv("JIUIN_RESOURCE_MANIFEST_JSON", "[]")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded with a production placeholder service token")
	}
}

func TestLoadRejectsDevelopmentSampleServiceTokenInProduction(t *testing.T) {
	t.Setenv("JIUIN_ENV", "production")
	t.Setenv("JIUIN_SERVER_HOST", "127.0.0.1")
	t.Setenv("JIUIN_SERVER_PORT", "8080")
	t.Setenv("JIUIN_SERVER_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("JIUIN_SERVER_READ_TIMEOUT", "15s")
	t.Setenv("JIUIN_SERVER_WRITE_TIMEOUT", "30s")
	t.Setenv("JIUIN_SERVER_IDLE_TIMEOUT", "60s")
	t.Setenv("JIUIN_SERVER_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("JIUIN_SITE_NAME", "Jiuin")
	t.Setenv("JIUIN_SITE_PROJECT", "Jiuin")
	t.Setenv("JIUIN_SITE_DOMAIN", "jiuin.cn")
	t.Setenv("JIUIN_MUSIC_DIRECTORY", "storage/music")
	t.Setenv("JIUIN_MUSIC_MAX_UPLOAD_SIZE", "100MB")
	t.Setenv("JIUIN_FFMPEG_PATH", "ffmpeg")
	t.Setenv("JIUIN_FFPROBE_PATH", "ffprobe")
	t.Setenv("JIUIN_MUSIC_FULL_BITRATE", "320k")
	t.Setenv("JIUIN_MUSIC_LITE_BITRATE", "128k")
	t.Setenv("JIUIN_MUSIC_OUTPUT_CODEC", "libmp3lame")
	t.Setenv("JIUIN_MUSIC_WORKER_COUNT", "2")
	t.Setenv("JIUIN_MUSIC_ADMIN_TOKEN", "development-music-admin-token-with-at-least-32-characters")
	t.Setenv("JIUIN_CORS_ALLOWED_ORIGINS", "https://jiuin.cn")
	t.Setenv("JIUIN_SHARED_SERVICE_TOKEN", "development-only-change-me-to-a-32-character-minimum-token")
	t.Setenv("JIUIN_MAIN_SITE_STATUS", "online")
	t.Setenv("JIUIN_BLOG_STATUS", "unknown")
	t.Setenv("JIUIN_EXTERNAL_LINKS_JSON", "[]")
	t.Setenv("JIUIN_RESOURCE_MANIFEST_JSON", "[]")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded with the development sample service token in production")
	}
}

func TestParseByteSizeRejectsMusicUploadLimitAboveConfiguredMaximum(t *testing.T) {
	value := "3GiB"
	parsed, err := parseByteSize(value)
	if err != nil {
		t.Fatalf("parseByteSize(%q): %v", value, err)
	}
	if parsed <= MaxMusicUploadSize {
		t.Fatalf("parsed size = %d, want a value above %d", parsed, MaxMusicUploadSize)
	}
}

func TestParseOptionalMusicProcessingTimeout(t *testing.T) {
	if value, err := parseOptionalMusicProcessingTimeout(""); err != nil || value != DefaultMusicProcessingTimeout {
		t.Fatalf("default processing timeout = (%v, %v), want (%v, nil)", value, err, DefaultMusicProcessingTimeout)
	}
	if _, err := parseOptionalMusicProcessingTimeout("25h"); err == nil {
		t.Fatal("accepted processing timeout beyond the maximum")
	}
}
