package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/core"
	"github.com/Joon-paxn/Jiuin/backend/internal/online"
)

func TestPublicContractUsesEnvelopeAndRelativeMusicURLs(t *testing.T) {
	storage := t.TempDir()
	config := core.Config{StorageDir: storage, DatabasePath: filepath.Join(storage, "music.db"), SiteName: "霁雪居", SiteProject: "Jiuin", SiteDomain: "jiuin.cn", AllowedOrigins: map[string]struct{}{"https://jiuin.cn": {}}}
	if err := config.EnsureStorage(); err != nil {
		t.Fatal(err)
	}
	db, err := core.OpenDatabase(config.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	handler := New(config, db, online.New(config.AllowedOrigins)).Handler()

	for _, route := range []string{"/api/v1/health", "/api/v1/site", "/api/v1/status", "/api/v1/music"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d", route, response.Code)
		}
		if !strings.Contains(response.Body.String(), `"code":200`) || !strings.Contains(response.Body.String(), `"message":"ok"`) {
			t.Fatalf("%s is not an API envelope: %s", route, response.Body.String())
		}
	}
}
