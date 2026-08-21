package api

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/core"
	"github.com/Joon-paxn/Jiuin/backend/internal/online"
)

type Server struct {
	config core.Config
	db     *sql.DB
	music  core.MusicStore
	online *online.Manager
}

func New(config core.Config, db *sql.DB, onlineManager *online.Manager) *Server {
	return &Server{config: config, db: db, music: core.MusicStore{DB: db, Config: config}, online: onlineManager}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("GET /api/v1/ready", s.ready)
	mux.HandleFunc("GET /api/v1/site", s.site)
	mux.HandleFunc("GET /api/v1/site/info", s.siteInfo)
	mux.HandleFunc("GET /api/v1/site/copyright", s.copyright)
	mux.HandleFunc("GET /api/v1/status", s.status)
	mux.HandleFunc("GET /api/v1/links", s.links)
	mux.HandleFunc("GET /api/v1/resources", s.resources)
	mux.HandleFunc("GET /api/v1/statistics", s.statistics)
	mux.HandleFunc("POST /api/v1/statistics/visit", s.recordVisit)
	mux.HandleFunc("GET /api/v1/background/random", s.randomBackground)
	mux.HandleFunc("GET /api/v1/music", s.listMusic)
	mux.HandleFunc("GET /api/v1/music/{id}", s.getMusic)
	mux.HandleFunc("POST /api/v1/admin/music/upload", s.uploadMusic)
	mux.HandleFunc("POST /api/v1/music/upload", s.uploadMusic)
	mux.HandleFunc("GET /media/music/{id}/{quality}", s.media)
	mux.HandleFunc("GET /ws/online", s.online.Handle)
	return withRequestID(mux)
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" || len(requestID) > 128 {
			requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, "ok", map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		respondError(w, http.StatusServiceUnavailable, "database is unavailable")
		return
	}
	for _, required := range []string{"tmp", "original", "full", "lite", "covers"} {
		if info, err := os.Stat(path.Join(s.config.StorageDir, required)); err != nil || !info.IsDir() {
			respondError(w, http.StatusServiceUnavailable, "storage is unavailable")
			return
		}
	}
	if err := toolReady(ctx, s.config.FFmpegPath); err != nil {
		respondError(w, http.StatusServiceUnavailable, "ffmpeg is unavailable")
		return
	}
	if err := toolReady(ctx, s.config.FFprobePath); err != nil {
		respondError(w, http.StatusServiceUnavailable, "ffprobe is unavailable")
		return
	}
	respond(w, http.StatusOK, "ready", map[string]any{"status": "ready", "dependencies": map[string]string{"database": "ok", "storage": "ok", "ffmpeg": "ok", "ffprobe": "ok"}})
}

func (s *Server) site(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, "ok", map[string]any{"site": siteData(s.config), "copyright": copyrightData(s.config)})
}
func (s *Server) siteInfo(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, "ok", siteData(s.config))
}
func (s *Server) copyright(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, "ok", copyrightData(s.config))
}
func siteData(c core.Config) map[string]string {
	return map[string]string{"name": c.SiteName, "project": c.SiteProject, "domain": c.SiteDomain}
}
func copyrightData(c core.Config) map[string]any {
	return map[string]any{"year": time.Now().Year(), "text": c.SiteProject}
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, "ok", map[string]any{"site": "online", "api": "online", "services": []map[string]string{}, "checkedAt": time.Now().UTC().Format(time.RFC3339Nano)})
}
func (s *Server) links(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, "ok", s.config.ExternalLinks)
}
func (s *Server) resources(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, "ok", s.config.Resources)
}

func (s *Server) statistics(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT path,views,last_visited_at FROM page_statistics ORDER BY path`)
	if err != nil {
		respondError(w, 500, "statistics are unavailable")
		return
	}
	defer rows.Close()
	type page struct {
		Path          string `json:"path"`
		Views         int64  `json:"views"`
		LastVisitedAt string `json:"lastVisitedAt"`
	}
	pages := []page{}
	var total int64
	for rows.Next() {
		var item page
		if err := rows.Scan(&item.Path, &item.Views, &item.LastVisitedAt); err != nil {
			respondError(w, 500, "statistics are unavailable")
			return
		}
		total += item.Views
		pages = append(pages, item)
	}
	respond(w, 200, "ok", map[string]any{"totalViews": total, "pages": pages})
}

func (s *Server) recordVisit(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r, s.config.ServiceToken) {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&input); err != nil || !strings.HasPrefix(input.Path, "/") || len(input.Path) > 512 {
		respondError(w, 400, "a valid path is required")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := coreWrite(r.Context(), s.db, `INSERT INTO page_statistics(path,views,last_visited_at) VALUES(?,1,?) ON CONFLICT(path) DO UPDATE SET views=views+1,last_visited_at=excluded.last_visited_at`, input.Path, now); err != nil {
		respondError(w, 503, "database is busy")
		return
	}
	respond(w, 200, "ok", map[string]string{"path": input.Path})
}

func (s *Server) randomBackground(w http.ResponseWriter, r *http.Request) {
	if len(s.config.BackgroundURLs) == 0 {
		respondError(w, http.StatusNotFound, "no background is configured")
		return
	}
	// Request ID ensures a stable but unimportant selection without tracking clients.
	index := 0
	for _, b := range []byte(r.Header.Get("X-Request-ID")) {
		index += int(b)
	}
	index %= len(s.config.BackgroundURLs)
	respond(w, 200, "ok", map[string]string{"url": s.config.BackgroundURLs[index]})
}

func (s *Server) listMusic(w http.ResponseWriter, r *http.Request) {
	items, err := s.music.ListPublic(r.Context())
	if err != nil {
		respondError(w, 500, "music is unavailable")
		return
	}
	respond(w, 200, "ok", items)
}
func (s *Server) getMusic(w http.ResponseWriter, r *http.Request) {
	item, err := s.music.GetPublic(r.Context(), r.PathValue("id"))
	if errors.Is(err, core.ErrNotFound) {
		respondError(w, 404, "music was not found")
		return
	}
	if err != nil {
		respondError(w, 500, "music is unavailable")
		return
	}
	respond(w, 200, "ok", item)
}

func (s *Server) uploadMusic(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r, s.config.AdminToken) {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		respondError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondError(w, http.StatusRequestEntityTooLarge, "upload exceeds the allowed size")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, 400, "file is required")
		return
	}
	defer file.Close()
	result, err := s.music.CreateUpload(r.Context(), core.UploadInput{IdempotencyKey: key, Title: r.FormValue("title"), Artist: r.FormValue("artist"), Album: r.FormValue("album"), AlbumArtist: r.FormValue("albumArtist"), Genre: r.FormValue("genre"), Year: r.FormValue("year"), SourceName: header.Filename, Source: file})
	if errors.Is(err, core.ErrIdempotencyKeyRequired) {
		respondError(w, 400, err.Error())
		return
	}
	if errors.Is(err, core.ErrIdempotencyConflict) {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "safe music") {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusServiceUnavailable, "upload is unavailable")
		return
	}
	status := http.StatusAccepted
	if result.IdempotentReplay {
		status = http.StatusOK
	}
	respond(w, status, "accepted", result)
}

func (s *Server) media(w http.ResponseWriter, r *http.Request) {
	file, contentType, err := s.music.OpenMedia(r.Context(), r.PathValue("id"), r.PathValue("quality"))
	if errors.Is(err, core.ErrNotFound) {
		respondError(w, 404, "media was not found")
		return
	}
	if err != nil {
		respondError(w, 500, "media is unavailable")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		respondError(w, 500, "media is unavailable")
		return
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func toolReady(ctx context.Context, tool string) error {
	return exec.CommandContext(ctx, tool, "-version").Run()
}

func (s *Server) authorized(r *http.Request, expected string) bool {
	if expected == "" {
		return false
	}
	provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return len(provided) == len(expected) && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func coreWrite(ctx context.Context, db *sql.DB, statement string, values ...any) error {
	// A single UPSERT is naturally short; retrying uses the same SQLite busy
	// semantics as the longer music transactions.
	for attempt := 0; attempt < 4; attempt++ {
		if _, err := db.ExecContext(ctx, statement, values...); err == nil {
			return nil
		}
	}
	return fmt.Errorf("write could not acquire SQLite")
}

type envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func respond(w http.ResponseWriter, status int, message string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Code: status, Message: message, Data: data})
}
func respondError(w http.ResponseWriter, status int, message string) {
	respond(w, status, message, nil)
}
