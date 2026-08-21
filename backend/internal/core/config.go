package core

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config is intentionally shared by the API, the backup worker, and the
// WebSocket server. PHP reads the same JIUIN_* variables from its FPM pool.
type Config struct {
	ListenAddr      string
	StorageDir      string
	DatabasePath    string
	FFmpegPath      string
	FFprobePath     string
	FullBitrate     string
	LiteBitrate     string
	OutputCodec     string
	AdminToken      string
	ServiceToken    string
	SiteName        string
	SiteProject     string
	SiteDomain      string
	ExternalLinks   []ExternalLink
	Resources       []Resource
	BackgroundURLs  []string
	AllowedOrigins  map[string]struct{}
	WorkerInterval  time.Duration
	ProcessingLease time.Duration
}

type ExternalLink struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type Resource struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Priority    int    `json:"priority"`
	CachePolicy string `json:"cachePolicy"`
}

func LoadConfig() (Config, error) {
	storage := env("JIUIN_STORAGE_DIR", "/var/lib/jiuin/music")
	database := env("JIUIN_DATABASE_PATH", filepath.Join(storage, "music.db"))
	links, err := parseJSON[[]ExternalLink](env("JIUIN_EXTERNAL_LINKS_JSON", "[]"))
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_EXTERNAL_LINKS_JSON: %w", err)
	}
	resources, err := parseJSON[[]Resource](env("JIUIN_RESOURCE_MANIFEST_JSON", "[]"))
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_RESOURCE_MANIFEST_JSON: %w", err)
	}
	backgrounds, err := parseJSON[[]string](env("JIUIN_BACKGROUND_URLS_JSON", "[]"))
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_BACKGROUND_URLS_JSON: %w", err)
	}
	for _, link := range links {
		if err := validatePublicURL(link.URL); err != nil {
			return Config{}, fmt.Errorf("external link URL: %w", err)
		}
	}
	for _, resource := range resources {
		if err := validatePublicURL(resource.URL); err != nil {
			return Config{}, fmt.Errorf("resource URL: %w", err)
		}
	}
	for _, background := range backgrounds {
		if err := validatePublicURL(background); err != nil {
			return Config{}, fmt.Errorf("background URL: %w", err)
		}
	}
	origins := make(map[string]struct{})
	for _, origin := range strings.Split(env("JIUIN_WS_ALLOWED_ORIGINS", "https://jiuin.cn,https://www.jiuin.cn"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}
	return Config{
		ListenAddr:     env("JIUIN_GO_LISTEN_ADDR", "127.0.0.1:8080"),
		StorageDir:     storage,
		DatabasePath:   database,
		FFmpegPath:     env("JIUIN_FFMPEG_PATH", "ffmpeg"),
		FFprobePath:    env("JIUIN_FFPROBE_PATH", "ffprobe"),
		FullBitrate:    env("JIUIN_MUSIC_FULL_BITRATE", "320k"),
		LiteBitrate:    env("JIUIN_MUSIC_LITE_BITRATE", "128k"),
		OutputCodec:    env("JIUIN_MUSIC_OUTPUT_CODEC", "libmp3lame"),
		AdminToken:     os.Getenv("JIUIN_MUSIC_ADMIN_TOKEN"),
		ServiceToken:   os.Getenv("JIUIN_SHARED_SERVICE_TOKEN"),
		SiteName:       env("JIUIN_SITE_NAME", "霁雪居"),
		SiteProject:    env("JIUIN_SITE_PROJECT", "Jiuin"),
		SiteDomain:     env("JIUIN_SITE_DOMAIN", "jiuin.cn"),
		ExternalLinks:  links,
		Resources:      resources,
		BackgroundURLs: backgrounds,
		AllowedOrigins: origins,
		WorkerInterval: durationEnv("JIUIN_MUSIC_WORKER_INTERVAL", 2*time.Second),
		// This must outlast normal FFmpeg work so an expired lease cannot make
		// PHP and Go process one task concurrently.
		ProcessingLease: durationEnv("JIUIN_MUSIC_PROCESSING_LEASE", 2*time.Hour),
	}, nil
}

func (c Config) EnsureStorage() error {
	for _, child := range []string{"", "tmp", "original", "full", "lite", "covers"} {
		if err := os.MkdirAll(filepath.Join(c.StorageDir, child), 0o750); err != nil {
			return err
		}
	}
	return nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseJSON[T any](value string) (T, error) {
	var parsed T
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return parsed, err
	}
	return parsed, nil
}

func validatePublicURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || value == "" || parsed.User != nil {
		return fmt.Errorf("must be a public URL")
	}
	if !parsed.IsAbs() {
		if !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(value, "//") {
			return fmt.Errorf("must be root-relative or HTTPS")
		}
		return nil
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" {
		return fmt.Errorf("must be root-relative or HTTPS")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "bkgapi.jiuin.cn" || net.ParseIP(host) != nil || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("must not expose an internal host")
	}
	return nil
}
