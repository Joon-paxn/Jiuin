package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Site        SiteConfig
	Music       MusicConfig
	CORS        CORSConfig
	Ecosystem   EcosystemConfig
}

type ServerConfig struct {
	Host              string
	Port              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func (config ServerConfig) Address() string {
	return net.JoinHostPort(config.Host, config.Port)
}

type SiteConfig struct {
	Name    string
	Project string
	Domain  string
}

type MusicConfig struct {
	// Directory is the private storage root. Public media is always served
	// through a Go/Nginx route; this filesystem path is never returned by an
	// API response.
	Directory         string
	MaxUploadSize     int64
	FFmpegPath        string
	FFprobePath       string
	FullBitrate       string
	LiteBitrate       string
	OutputCodec       string
	WorkerCount       int
	ProcessingTimeout time.Duration
	AdminToken        string
}

// MaxMusicUploadSize keeps the HTTP and service-side "+1 byte" sentinel
// arithmetic safe and prevents a configuration typo from reserving an
// impractical amount of disk/CPU work per upload.
const MaxMusicUploadSize int64 = 2 << 30

const DefaultMusicProcessingTimeout = 2 * time.Hour

type CORSConfig struct {
	AllowedOrigins []string
}

type EcosystemConfig struct {
	SharedServiceToken string
	MainSiteStatus     string
	BlogStatus         string
	ExternalLinks      []ExternalLinkConfig
	Resources          []ResourceConfig
}

type ExternalLinkConfig struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type ResourceConfig struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Priority    int    `json:"priority"`
	CachePolicy string `json:"cachePolicy"`
}

// Load reads service settings from environment variables so configuration never ships in code.
func Load() (Config, error) {
	values, err := requiredValues(
		"JIUIN_ENV",
		"JIUIN_SERVER_HOST",
		"JIUIN_SERVER_PORT",
		"JIUIN_SERVER_READ_HEADER_TIMEOUT",
		"JIUIN_SERVER_READ_TIMEOUT",
		"JIUIN_SERVER_WRITE_TIMEOUT",
		"JIUIN_SERVER_IDLE_TIMEOUT",
		"JIUIN_SERVER_SHUTDOWN_TIMEOUT",
		"JIUIN_SITE_NAME",
		"JIUIN_SITE_PROJECT",
		"JIUIN_SITE_DOMAIN",
		"JIUIN_MUSIC_DIRECTORY",
		"JIUIN_MUSIC_MAX_UPLOAD_SIZE",
		"JIUIN_FFMPEG_PATH",
		"JIUIN_FFPROBE_PATH",
		"JIUIN_MUSIC_FULL_BITRATE",
		"JIUIN_MUSIC_LITE_BITRATE",
		"JIUIN_MUSIC_OUTPUT_CODEC",
		"JIUIN_MUSIC_WORKER_COUNT",
		"JIUIN_MUSIC_ADMIN_TOKEN",
		"JIUIN_CORS_ALLOWED_ORIGINS",
		"JIUIN_SHARED_SERVICE_TOKEN",
		"JIUIN_MAIN_SITE_STATUS",
		"JIUIN_BLOG_STATUS",
		"JIUIN_EXTERNAL_LINKS_JSON",
		"JIUIN_RESOURCE_MANIFEST_JSON",
	)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := time.ParseDuration(values["JIUIN_SERVER_READ_HEADER_TIMEOUT"])
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_SERVER_READ_HEADER_TIMEOUT must be a duration: %w", err)
	}
	readTimeout, err := time.ParseDuration(values["JIUIN_SERVER_READ_TIMEOUT"])
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_SERVER_READ_TIMEOUT must be a duration: %w", err)
	}
	writeTimeout, err := time.ParseDuration(values["JIUIN_SERVER_WRITE_TIMEOUT"])
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_SERVER_WRITE_TIMEOUT must be a duration: %w", err)
	}
	idleTimeout, err := time.ParseDuration(values["JIUIN_SERVER_IDLE_TIMEOUT"])
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_SERVER_IDLE_TIMEOUT must be a duration: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(values["JIUIN_SERVER_SHUTDOWN_TIMEOUT"])
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_SERVER_SHUTDOWN_TIMEOUT must be a duration: %w", err)
	}

	if err := validateServerTimeouts(readHeaderTimeout, readTimeout, writeTimeout, idleTimeout, shutdownTimeout); err != nil {
		return Config{}, err
	}

	environment := strings.ToLower(values["JIUIN_ENV"])
	if environment != "development" && environment != "production" {
		return Config{}, fmt.Errorf("JIUIN_ENV must be development or production")
	}

	allowedOrigins, err := parseAllowedOrigins(values["JIUIN_CORS_ALLOWED_ORIGINS"], environment)
	if err != nil {
		return Config{}, err
	}
	if err := validateSharedServiceToken(values["JIUIN_SHARED_SERVICE_TOKEN"], environment); err != nil {
		return Config{}, err
	}
	if !isKnownStatus(values["JIUIN_MAIN_SITE_STATUS"]) || !isKnownStatus(values["JIUIN_BLOG_STATUS"]) {
		return Config{}, fmt.Errorf("ecosystem status must be online, degraded, offline, or unknown")
	}
	maxUploadSize, err := parseByteSize(values["JIUIN_MUSIC_MAX_UPLOAD_SIZE"])
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_MUSIC_MAX_UPLOAD_SIZE: %w", err)
	}
	if maxUploadSize > MaxMusicUploadSize {
		return Config{}, fmt.Errorf("JIUIN_MUSIC_MAX_UPLOAD_SIZE must not exceed %d bytes", MaxMusicUploadSize)
	}
	workerCount, err := strconv.Atoi(values["JIUIN_MUSIC_WORKER_COUNT"])
	if err != nil || workerCount < 1 || workerCount > 32 {
		return Config{}, fmt.Errorf("JIUIN_MUSIC_WORKER_COUNT must be an integer between 1 and 32")
	}
	processingTimeout, err := parseOptionalMusicProcessingTimeout(os.Getenv("JIUIN_MUSIC_PROCESSING_TIMEOUT"))
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_MUSIC_PROCESSING_TIMEOUT: %w", err)
	}
	if !validBitrate(values["JIUIN_MUSIC_FULL_BITRATE"]) || !validBitrate(values["JIUIN_MUSIC_LITE_BITRATE"]) {
		return Config{}, fmt.Errorf("music bitrates must use a positive kbps value such as 320k")
	}
	if bitrateKbps(values["JIUIN_MUSIC_FULL_BITRATE"]) < bitrateKbps(values["JIUIN_MUSIC_LITE_BITRATE"]) {
		return Config{}, fmt.Errorf("JIUIN_MUSIC_FULL_BITRATE must not be lower than JIUIN_MUSIC_LITE_BITRATE")
	}
	if !validExecutableSetting(values["JIUIN_FFMPEG_PATH"]) || !validExecutableSetting(values["JIUIN_FFPROBE_PATH"]) {
		return Config{}, fmt.Errorf("FFmpeg executable settings must not be empty or contain a NUL byte")
	}
	if !validCodecName(values["JIUIN_MUSIC_OUTPUT_CODEC"]) {
		return Config{}, fmt.Errorf("JIUIN_MUSIC_OUTPUT_CODEC contains unsupported characters")
	}
	if err := validateAdminToken(values["JIUIN_MUSIC_ADMIN_TOKEN"], environment); err != nil {
		return Config{}, err
	}

	var externalLinks []ExternalLinkConfig
	if err := parseJSONList("JIUIN_EXTERNAL_LINKS_JSON", values["JIUIN_EXTERNAL_LINKS_JSON"], &externalLinks); err != nil {
		return Config{}, err
	}

	var resources []ResourceConfig
	if err := parseJSONList("JIUIN_RESOURCE_MANIFEST_JSON", values["JIUIN_RESOURCE_MANIFEST_JSON"], &resources); err != nil {
		return Config{}, err
	}

	return Config{
		Environment: environment,
		Server: ServerConfig{
			Host:              values["JIUIN_SERVER_HOST"],
			Port:              values["JIUIN_SERVER_PORT"],
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ShutdownTimeout:   shutdownTimeout,
		},
		Site: SiteConfig{
			Name:    values["JIUIN_SITE_NAME"],
			Project: values["JIUIN_SITE_PROJECT"],
			Domain:  values["JIUIN_SITE_DOMAIN"],
		},
		Music: MusicConfig{
			Directory:         values["JIUIN_MUSIC_DIRECTORY"],
			MaxUploadSize:     maxUploadSize,
			FFmpegPath:        values["JIUIN_FFMPEG_PATH"],
			FFprobePath:       values["JIUIN_FFPROBE_PATH"],
			FullBitrate:       values["JIUIN_MUSIC_FULL_BITRATE"],
			LiteBitrate:       values["JIUIN_MUSIC_LITE_BITRATE"],
			OutputCodec:       values["JIUIN_MUSIC_OUTPUT_CODEC"],
			WorkerCount:       workerCount,
			ProcessingTimeout: processingTimeout,
			AdminToken:        values["JIUIN_MUSIC_ADMIN_TOKEN"],
		},
		CORS: CORSConfig{AllowedOrigins: allowedOrigins},
		Ecosystem: EcosystemConfig{
			SharedServiceToken: values["JIUIN_SHARED_SERVICE_TOKEN"],
			MainSiteStatus:     values["JIUIN_MAIN_SITE_STATUS"],
			BlogStatus:         values["JIUIN_BLOG_STATUS"],
			ExternalLinks:      externalLinks,
			Resources:          resources,
		},
	}, nil
}

func parseOptionalMusicProcessingTimeout(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultMusicProcessingTimeout, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 || parsed > 24*time.Hour {
		return 0, fmt.Errorf("must be a duration greater than zero and no more than 24h")
	}
	return parsed, nil
}

func validateServerTimeouts(readHeader, read, write, idle, shutdown time.Duration) error {
	for name, value := range map[string]time.Duration{
		"JIUIN_SERVER_READ_HEADER_TIMEOUT": readHeader,
		"JIUIN_SERVER_READ_TIMEOUT":        read,
		"JIUIN_SERVER_WRITE_TIMEOUT":       write,
		"JIUIN_SERVER_IDLE_TIMEOUT":        idle,
		"JIUIN_SERVER_SHUTDOWN_TIMEOUT":    shutdown,
	} {
		if value <= 0 {
			return fmt.Errorf("%s must be greater than zero", name)
		}
	}

	return nil
}

func parseAllowedOrigins(value, environment string) ([]string, error) {
	origins := splitList(value)
	if len(origins) == 0 {
		return nil, fmt.Errorf("JIUIN_CORS_ALLOWED_ORIGINS must contain at least one origin")
	}

	seen := make(map[string]struct{}, len(origins))
	validated := make([]string, 0, len(origins))
	for _, origin := range origins {
		parsed, err := url.ParseRequestURI(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, fmt.Errorf("JIUIN_CORS_ALLOWED_ORIGINS contains an invalid origin")
		}
		if strings.EqualFold(environment, "production") && parsed.Scheme != "https" {
			return nil, fmt.Errorf("JIUIN_CORS_ALLOWED_ORIGINS must use HTTPS in production")
		}

		normalized := parsed.String()
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		validated = append(validated, normalized)
	}

	return validated, nil
}

func validateSharedServiceToken(token, environment string) error {
	return validateSecret("JIUIN_SHARED_SERVICE_TOKEN", token, environment)
}

func validateAdminToken(token, environment string) error {
	return validateSecret("JIUIN_MUSIC_ADMIN_TOKEN", token, environment)
}

func validateSecret(name, token, environment string) error {
	if len(token) < 32 {
		return fmt.Errorf("%s must contain at least 32 characters", name)
	}
	if strings.EqualFold(environment, "production") {
		placeholderMarkers := []string{
			"replace-with",
			"development-only",
			"change-me",
			"example",
		}
		lowerToken := strings.ToLower(token)
		for _, marker := range placeholderMarkers {
			if strings.Contains(lowerToken, marker) {
				return fmt.Errorf("%s must be replaced in production", name)
			}
		}
	}

	return nil
}

var bitratePattern = regexp.MustCompile(`^[1-9][0-9]{0,4}k$`)
var codecNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func validBitrate(value string) bool {
	return bitratePattern.MatchString(strings.ToLower(strings.TrimSpace(value)))
}

func bitrateKbps(value string) int {
	number := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), "k")
	parsed, _ := strconv.Atoi(number)
	return parsed
}

func validExecutableSetting(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.ContainsRune(value, '\x00')
}

func validCodecName(value string) bool {
	return codecNamePattern.MatchString(strings.TrimSpace(value))
}

func parseByteSize(value string) (int64, error) {
	text := strings.ToUpper(strings.TrimSpace(value))
	if text == "" {
		return 0, fmt.Errorf("must not be empty")
	}

	multipliers := []struct {
		suffix     string
		multiplier int64
	}{
		{"MIB", 1024 * 1024},
		{"GIB", 1024 * 1024 * 1024},
		{"KIB", 1024},
		{"MB", 1000 * 1000},
		{"GB", 1000 * 1000 * 1000},
		{"KB", 1000},
		{"B", 1},
	}

	multiplier := int64(1)
	for _, candidate := range multipliers {
		if strings.HasSuffix(text, candidate.suffix) {
			text = strings.TrimSpace(strings.TrimSuffix(text, candidate.suffix))
			multiplier = candidate.multiplier
			break
		}
	}

	amount, err := strconv.ParseInt(text, 10, 64)
	if err != nil || amount <= 0 || amount > (1<<63-1)/multiplier {
		return 0, fmt.Errorf("must be a positive byte value such as 50MiB")
	}

	return amount * multiplier, nil
}

func isKnownStatus(value string) bool {
	return value == "online" || value == "degraded" || value == "offline" || value == "unknown"
}

func parseJSONList(name, value string, destination any) error {
	if err := json.Unmarshal([]byte(value), destination); err != nil {
		return fmt.Errorf("%s must be a valid JSON array: %w", name, err)
	}
	return nil
}

func requiredValues(keys ...string) (map[string]string, error) {
	values := make(map[string]string, len(keys))

	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			return nil, missingConfigurationError{key: key}
		}

		values[key] = value
	}

	return values, nil
}

type missingConfigurationError struct {
	key string
}

func (error missingConfigurationError) Error() string {
	return fmt.Sprintf("required environment variable %s is not set", error.key)
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))

	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			items = append(items, item)
		}
	}

	return items
}
