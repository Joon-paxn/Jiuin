package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/handler"
	"github.com/Joon-paxn/Jiuin/backend/internal/api/middleware"
	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
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
	getOrHead := middleware.RequireMethods(http.MethodGet, http.MethodHead)

	mux.Handle("/api/v1/health", getOrHead(http.HandlerFunc(handler.Health)))
	mux.Handle("/api/v1/site/info", getOrHead(http.HandlerFunc(siteHandler.Info)))
	mux.Handle("/api/v1/site/copyright", getOrHead(http.HandlerFunc(siteHandler.Copyright)))
	mux.Handle("/api/v1/site", getOrHead(http.HandlerFunc(siteHandler.Shared)))
	mux.Handle("/api/v1/music/list", getOrHead(http.HandlerFunc(musicHandler.List)))
	mux.Handle(
		"/media/music/{id}",
		middleware.RequireMethods(http.MethodGet, http.MethodHead)(
			middleware.RateLimit(120, time.Minute)(http.HandlerFunc(musicHandler.Stream)),
		),
	)
	mux.Handle("/api/v1/statistics", getOrHead(http.HandlerFunc(statisticsHandler.Summary)))
	mux.Handle(
		"/api/v1/statistics/visit",
		middleware.RequireMethods(http.MethodPost)(
			middleware.RateLimit(30, time.Minute)(
				middleware.RequireServiceToken(serviceToken)(http.HandlerFunc(statisticsHandler.Record)),
			),
		),
	)
	mux.Handle("/api/v1/status", getOrHead(http.HandlerFunc(statusHandler.Get)))
	mux.Handle("/api/v1/links", getOrHead(http.HandlerFunc(linkHandler.List)))
	mux.Handle("/api/v1/resources", getOrHead(http.HandlerFunc(resourceHandler.List)))
	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusNotFound, "API route was not found")
	}))
	mux.Handle("/media/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusNotFound, "media route was not found")
	}))

	return middleware.RequestID(
		middleware.RequestLogger(logger)(
			middleware.Recovery(logger)(
				middleware.SecurityHeaders(
					middleware.CORS(allowedOrigins)(
						// Count every public request before selecting a route so an
						// attacker cannot bypass the broad edge guard via misses.
						middleware.RateLimit(360, time.Minute)(
							middleware.AllowMethods(http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions)(mux),
						),
					),
				),
			),
		),
	)
}
