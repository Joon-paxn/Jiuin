package config

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Site        SiteConfig
	CORS        CORSConfig
}

type ServerConfig struct {
	Host              string
	Port              string
	ReadHeaderTimeout time.Duration
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

type CORSConfig struct {
	AllowedOrigins []string
}

// Load reads service settings from environment variables so configuration never ships in code.
func Load() (Config, error) {
	values, err := requiredValues(
		"JIUIN_ENV",
		"JIUIN_SERVER_HOST",
		"JIUIN_SERVER_PORT",
		"JIUIN_SERVER_READ_HEADER_TIMEOUT",
		"JIUIN_SERVER_SHUTDOWN_TIMEOUT",
		"JIUIN_SITE_NAME",
		"JIUIN_SITE_PROJECT",
		"JIUIN_SITE_DOMAIN",
		"JIUIN_CORS_ALLOWED_ORIGINS",
	)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := time.ParseDuration(values["JIUIN_SERVER_READ_HEADER_TIMEOUT"])
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_SERVER_READ_HEADER_TIMEOUT must be a duration: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(values["JIUIN_SERVER_SHUTDOWN_TIMEOUT"])
	if err != nil {
		return Config{}, fmt.Errorf("JIUIN_SERVER_SHUTDOWN_TIMEOUT must be a duration: %w", err)
	}

	allowedOrigins := splitList(values["JIUIN_CORS_ALLOWED_ORIGINS"])
	if len(allowedOrigins) == 0 {
		return Config{}, fmt.Errorf("JIUIN_CORS_ALLOWED_ORIGINS must contain at least one origin")
	}

	return Config{
		Environment: values["JIUIN_ENV"],
		Server: ServerConfig{
			Host:              values["JIUIN_SERVER_HOST"],
			Port:              values["JIUIN_SERVER_PORT"],
			ReadHeaderTimeout: readHeaderTimeout,
			ShutdownTimeout:   shutdownTimeout,
		},
		Site: SiteConfig{
			Name:    values["JIUIN_SITE_NAME"],
			Project: values["JIUIN_SITE_PROJECT"],
			Domain:  values["JIUIN_SITE_DOMAIN"],
		},
		CORS: CORSConfig{AllowedOrigins: allowedOrigins},
	}, nil
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
