package api

import (
	"log/slog"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/handler"
	"github.com/Joon-paxn/Jiuin/backend/internal/api/middleware"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

func NewRouter(siteService service.SiteService, musicService service.MusicService, allowedOrigins []string, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	siteHandler := handler.NewSiteHandler(siteService, logger)
	musicHandler := handler.NewMusicHandler(musicService, logger)

	mux.HandleFunc("GET /api/v1/health", handler.Health)
	mux.HandleFunc("GET /api/v1/site/info", siteHandler.Info)
	mux.HandleFunc("GET /api/v1/site/copyright", siteHandler.Copyright)
	mux.HandleFunc("GET /api/v1/music/list", musicHandler.List)

	return middleware.Recovery(logger)(
		middleware.RequestLogger(logger)(
			middleware.CORS(allowedOrigins)(mux),
		),
	)
}
