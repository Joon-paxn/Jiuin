package api

import (
	"log/slog"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/handler"
	"github.com/Joon-paxn/Jiuin/backend/internal/api/middleware"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

func NewRouter(
	siteService service.SiteService,
	musicService service.MusicService,
	statisticsService service.StatisticsService,
	statusService service.StatusService,
	linkService service.LinkService,
	resourceService service.ResourceService,
	serviceToken string,
	allowedOrigins []string,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()
	siteHandler := handler.NewSiteHandler(siteService, logger)
	musicHandler := handler.NewMusicHandler(musicService, logger)
	statisticsHandler := handler.NewStatisticsHandler(statisticsService, logger)
	statusHandler := handler.NewStatusHandler(statusService, logger)
	linkHandler := handler.NewLinkHandler(linkService, logger)
	resourceHandler := handler.NewResourceHandler(resourceService, logger)

	mux.HandleFunc("GET /api/v1/health", handler.Health)
	mux.HandleFunc("GET /api/v1/site/info", siteHandler.Info)
	mux.HandleFunc("GET /api/v1/site/copyright", siteHandler.Copyright)
	mux.HandleFunc("GET /api/v1/site", siteHandler.Shared)
	mux.HandleFunc("GET /api/v1/music/list", musicHandler.List)
	mux.HandleFunc("GET /api/v1/statistics", statisticsHandler.Summary)
	mux.Handle("POST /api/v1/statistics/visit", middleware.RequireServiceToken(serviceToken)(http.HandlerFunc(statisticsHandler.Record)))
	mux.HandleFunc("GET /api/v1/status", statusHandler.Get)
	mux.HandleFunc("GET /api/v1/links", linkHandler.List)
	mux.HandleFunc("GET /api/v1/resources", resourceHandler.List)

	return middleware.Recovery(logger)(
		middleware.RequestLogger(logger)(
			middleware.CORS(allowedOrigins)(mux),
		),
	)
}
