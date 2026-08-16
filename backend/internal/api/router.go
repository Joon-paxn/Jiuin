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
	musicAdminToken string,
	allowedOrigins []string,
	logger *slog.Logger,
) http.Handler {
	return NewRouterWithHTTPMask(siteService, musicService, statisticsService, statusService, linkService, resourceService, serviceToken, musicAdminToken, allowedOrigins, false, http.StatusTeapot, logger, func() bool { return true })
}

// NewRouterWithReadiness composes the same public API as NewRouter while
// allowing the application lifecycle to expose a truthful readiness state.
func NewRouterWithReadiness(
	siteService service.SiteService,
	musicService service.MusicService,
	statisticsService service.StatisticsService,
	statusService service.StatusService,
	linkService service.LinkService,
	resourceService service.ResourceService,
	serviceToken string,
	musicAdminToken string,
	allowedOrigins []string,
	logger *slog.Logger,
	ready func() bool,
) http.Handler {
	return NewRouterWithHTTPMask(siteService, musicService, statisticsService, statusService, linkService, resourceService, serviceToken, musicAdminToken, allowedOrigins, false, http.StatusTeapot, logger, ready)
}

// NewRouterWithHTTPMask exposes only the configured successful JSON routes as
// HTTP 418. Existing constructors keep masking disabled for tests and local
// development unless the application explicitly enables it through config.
func NewRouterWithHTTPMask(
	siteService service.SiteService,
	musicService service.MusicService,
	statisticsService service.StatisticsService,
	statusService service.StatusService,
	linkService service.LinkService,
	resourceService service.ResourceService,
	serviceToken string,
	musicAdminToken string,
	allowedOrigins []string,
	httpMaskEnabled bool,
	httpMaskStatus int,
	logger *slog.Logger,
	ready func() bool,
) http.Handler {
	mux := http.NewServeMux()
	backgroundHandler := handler.NewBackgroundHandler(logger)
	siteHandler := handler.NewSiteHandler(siteService, logger)
	musicHandler := handler.NewMusicHandler(musicService, logger)
	var managedMusicHandler handler.MusicHandler
	hasManagedMusic := false
	if processingService, ok := musicService.(service.MusicProcessingService); ok {
		// The handler gets the configured limit from the processing service at
		// composition time in main. Tests with legacy services remain simple.
		managedMusicHandler = handler.NewManagedMusicHandler(processingService, processingService.MaxUploadSize(), logger)
		hasManagedMusic = true
	}
	statisticsHandler := handler.NewStatisticsHandler(statisticsService, logger)
	statusHandler := handler.NewStatusHandler(statusService, logger)
	linkHandler := handler.NewLinkHandler(linkService, logger)
	resourceHandler := handler.NewResourceHandler(resourceService, logger)
	getOrHead := middleware.RequireMethods(http.MethodGet, http.MethodHead)

	mux.Handle("/health", getOrHead(http.HandlerFunc(handler.Health)))
	mux.Handle("/ready", getOrHead(handler.Ready(ready)))
	mux.Handle("/api/v1/health", getOrHead(http.HandlerFunc(handler.Health)))
	mux.Handle("/api/v1/ready", getOrHead(handler.Ready(ready)))
	mux.Handle("/api/v1/background/random", getOrHead(http.HandlerFunc(backgroundHandler.Random)))
	mux.Handle("/api/v1/site/info", getOrHead(http.HandlerFunc(siteHandler.Info)))
	mux.Handle("/api/v1/site/copyright", getOrHead(http.HandlerFunc(siteHandler.Copyright)))
	mux.Handle("/api/v1/site", getOrHead(http.HandlerFunc(siteHandler.Shared)))
	if hasManagedMusic {
		// /music/list stays as a backwards-compatible view for clients deployed
		// before /api/v1/music. It reads the same SQLite-backed library.
		mux.Handle("/api/v1/music/list", getOrHead(http.HandlerFunc(musicHandler.List)))
		mux.Handle("/api/v1/music", getOrHead(http.HandlerFunc(managedMusicHandler.ListPublic)))
		mux.Handle("/api/v1/music/tasks/{task_id}", getOrHead(http.HandlerFunc(managedMusicHandler.Task)))
		mux.Handle("/api/v1/music/{id}", getOrHead(http.HandlerFunc(managedMusicHandler.GetPublic)))
		mux.Handle(
			"/api/v1/admin/music/upload",
			middleware.RequireMethods(http.MethodPost)(
				middleware.RequireServiceToken(musicAdminToken)(
					middleware.RateLimit(3, time.Minute)(
						http.HandlerFunc(managedMusicHandler.Upload),
					),
				),
			),
		)
		// The initial requested alias is protected as well. The admin-prefixed
		// route is canonical, but neither spelling becomes public accidentally.
		mux.Handle(
			"/api/v1/music/upload",
			middleware.RequireMethods(http.MethodPost)(
				middleware.RequireServiceToken(musicAdminToken)(
					middleware.RateLimit(3, time.Minute)(
						http.HandlerFunc(managedMusicHandler.Upload),
					),
				),
			),
		)
		mux.Handle(
			"/media/music/{variant}/{file}",
			middleware.RequireMethods(http.MethodGet, http.MethodHead)(
				middleware.RateLimit(120, time.Minute)(http.HandlerFunc(managedMusicHandler.StreamManaged)),
			),
		)
	} else {
		mux.Handle("/api/v1/music/list", getOrHead(http.HandlerFunc(musicHandler.List)))
	}
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
			middleware.RequireServiceToken(serviceToken)(
				middleware.RateLimit(30, time.Minute)(
					http.HandlerFunc(statisticsHandler.Record),
				),
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

	maskedJSONRoutes := middleware.MaskSuccessfulJSON(middleware.JSONStatusMaskConfig{
		Enabled: httpMaskEnabled,
		Status:  httpMaskStatus,
		Routes:  []string{"/api/v1/music", "/api/v1/music/{id}"},
	})(mux)

	return middleware.RequestID(
		middleware.RequestLogger(logger)(
			middleware.Recovery(logger)(
				middleware.SecurityHeaders(
					// Count every request before CORS or route selection. This makes
					// malformed and rejected origins consume a bounded share too.
					middleware.RateLimit(360, time.Minute)(
						middleware.CORS(allowedOrigins)(
							middleware.AllowMethods(http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions)(maskedJSONRoutes),
						),
					),
				),
			),
		),
	)
}
