package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type MusicHandler struct {
	service service.MusicService
	logger  *slog.Logger
}

func NewMusicHandler(service service.MusicService, logger *slog.Logger) MusicHandler {
	return MusicHandler{service: service, logger: logger}
}

func (handler MusicHandler) List(w http.ResponseWriter, r *http.Request) {
	tracks, err := handler.service.List(r.Context())
	if err != nil {
		handler.logger.Error("failed to load music list", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load music list")
		return
	}

	tracks = append([]model.MusicTrack(nil), tracks...)
	for index := range tracks {
		tracks[index].SourceURL = publicMusicURL(r, tracks[index].SourceURL)
	}

	response.Success(w, tracks)
}

// Stream serves a resolved local track by opaque ID. http.ServeContent provides
// seek/range support required by the browser audio element.
func (handler MusicHandler) Stream(w http.ResponseWriter, r *http.Request) {
	// Audio playback may legitimately outlast the API's normal write timeout.
	// Clear only this response deadline after routing/auth/rate-limit checks;
	// regular API endpoints retain the server-wide write deadline.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		handler.logger.Warn("failed to clear music stream write deadline", "error", err)
	}

	asset, err := handler.service.Open(r.Context(), r.PathValue("id"))
	if errors.Is(err, service.ErrMusicNotFound) {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to resolve music track", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to resolve music track")
		return
	}

	beforeOpen, err := os.Lstat(asset.Path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && !beforeOpen.Mode().IsRegular()) {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to inspect music track", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to inspect music track")
		return
	}

	file, err := os.Open(asset.Path)
	if errors.Is(err, os.ErrNotExist) {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to open music track", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to open music track")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(beforeOpen, info) {
		if err != nil {
			handler.logger.Error("failed to inspect music track", "error", err)
		}
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
	w.Header().Set("ETag", fmt.Sprintf(`W/"%x-%x"`, info.Size(), info.ModTime().UnixNano()))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if contentType := audioContentType(asset.Name); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, asset.Name, info.ModTime(), file)
}

func publicMusicURL(r *http.Request, sourceURL string) string {
	if !strings.HasPrefix(sourceURL, "/media/music/") || r.Host == "" {
		return sourceURL
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if isTrustedLoopbackPeer(r.RemoteAddr) && strings.EqualFold(strings.TrimSpace(strings.SplitN(r.Header.Get("X-Forwarded-Proto"), ",", 2)[0]), "https") {
		// Only Nginx on the local loopback interface is allowed to influence
		// public URL generation. A direct client cannot force arbitrary proxy
		// semantics by sending X-Forwarded-Proto itself.
		scheme = "https"
	}

	return (&url.URL{Scheme: scheme, Host: r.Host, Path: sourceURL}).String()
}

func isTrustedLoopbackPeer(remoteAddress string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddress))
	if err != nil {
		host = strings.TrimSpace(remoteAddress)
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func audioContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".aac":
		return "audio/aac"
	case ".flac":
		return "audio/flac"
	case ".m4a":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	default:
		return ""
	}
}
