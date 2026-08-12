package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
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
	Directory string
}

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
		Music: MusicConfig{Directory: values["JIUIN_MUSIC_DIRECTORY"]},
		CORS:  CORSConfig{AllowedOrigins: allowedOrigins},
		Ecosystem: EcosystemConfig{
			SharedServiceToken: values["JIUIN_SHARED_SERVICE_TOKEN"],
			MainSiteStatus:     values["JIUIN_MAIN_SITE_STATUS"],
			BlogStatus:         values["JIUIN_BLOG_STATUS"],
			ExternalLinks:      externalLinks,
			Resources:          resources,
		},
	}, nil
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
	if len(token) < 32 {
		return fmt.Errorf("JIUIN_SHARED_SERVICE_TOKEN must contain at least 32 characters")
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
				return fmt.Errorf("JIUIN_SHARED_SERVICE_TOKEN must be replaced in production")
			}
		}
	}

	return nil
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
