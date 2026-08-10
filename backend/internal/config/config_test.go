package config

import "testing"

func TestLoadReadsRequiredEnvironmentConfiguration(t *testing.T) {
	t.Setenv("JIUIN_ENV", "test")
	t.Setenv("JIUIN_SERVER_HOST", "127.0.0.1")
	t.Setenv("JIUIN_SERVER_PORT", "8080")
	t.Setenv("JIUIN_SERVER_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("JIUIN_SERVER_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("JIUIN_SITE_NAME", "霁雪居")
	t.Setenv("JIUIN_SITE_PROJECT", "Jiuin")
	t.Setenv("JIUIN_SITE_DOMAIN", "Jiuin.cn")
	t.Setenv("JIUIN_CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://jiuin.cn")

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
}
