package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/api"
	"github.com/Joon-paxn/Jiuin/backend/internal/config"
	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

// Application owns the process-wide dependencies and their lifecycle. All
// construction happens once; handlers and workers receive the shared services.
type Application struct {
	config config.Config
	logger *slog.Logger
	server *http.Server
	music  service.MusicProcessingService
	closer interface{ Close() error }

	ready        atomic.Bool
	shutdownOnce sync.Once
}

func New(cfg config.Config, logger *slog.Logger) (*Application, error) {
	if logger == nil {
		logger = slog.Default()
	}
	bootedAt := time.Now()
	logger.Info("[BOOT] starting Jiuin Backend", "environment", cfg.Environment)

	stageStartedAt := time.Now()
	musicRepository, err := repository.NewSQLiteMusicRepository(cfg.Music.Directory)
	if err != nil {
		return nil, fmt.Errorf("initialize music database and storage: %w", err)
	}
	closer, ok := musicRepository.(interface{ Close() error })
	if !ok {
		return nil, errors.New("music repository does not expose Close")
	}
	logger.Info("[DATABASE] ready", "duration", time.Since(stageStartedAt), "storage", cfg.Music.Directory)

	musicService := service.NewMusicProcessingService(musicRepository, cfg.Music, logger)
	siteService := service.NewSiteService(repository.NewConfigSiteRepository(model.SiteInfo{Name: cfg.Site.Name, Project: cfg.Site.Project, Domain: cfg.Site.Domain}), cfg.Site.Domain)
	statisticsService := service.NewStatisticsService(repository.NewMemoryStatisticsRepository())
	statusService := service.NewStatusService(repository.NewStaticStatusRepository(model.EcosystemStatus{
		Site: cfg.Ecosystem.MainSiteStatus, API: "online",
		Services: []model.ServiceStatus{{Name: "main-site", Status: cfg.Ecosystem.MainSiteStatus}, {Name: "blog", Status: cfg.Ecosystem.BlogStatus}, {Name: "api", Status: "online"}},
	}))
	links := make([]model.ExternalLink, 0, len(cfg.Ecosystem.ExternalLinks))
	for _, link := range cfg.Ecosystem.ExternalLinks {
		links = append(links, model.ExternalLink{Name: link.Name, URL: link.URL, Description: link.Description})
	}
	resources := make([]model.ResourceDescriptor, 0, len(cfg.Ecosystem.Resources))
	for _, resource := range cfg.Ecosystem.Resources {
		resources = append(resources, model.ResourceDescriptor{Name: resource.Name, URL: resource.URL, Priority: resource.Priority, CachePolicy: resource.CachePolicy})
	}

	application := &Application{config: cfg, logger: logger, music: musicService, closer: closer}
	application.server = &http.Server{
		Addr:              cfg.Server.Address(),
		Handler:           api.NewRouterWithHTTPMask(siteService, musicService, statisticsService, statusService, service.NewLinkService(repository.NewStaticLinkRepository(links)), service.NewResourceService(repository.NewStaticResourceRepository(resources)), cfg.Ecosystem.SharedServiceToken, cfg.Music.AdminToken, cfg.CORS.AllowedOrigins, cfg.HTTPMask.Enabled, cfg.HTTPMask.Status, logger, application.Ready),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout, ReadTimeout: cfg.Server.ReadTimeout, WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout: cfg.Server.IdleTimeout, MaxHeaderBytes: 1 << 20,
	}
	logger.Info("[CORE] ready", "duration", time.Since(stageStartedAt))
	logger.Info("[BOOT] dependencies initialized", "duration", time.Since(bootedAt))
	return application, nil
}

func (application *Application) Ready() bool { return application.ready.Load() }

// Run binds HTTP before publishing readiness. Worker recovery is intentionally
// asynchronous inside MusicProcessingService and never delays this transition.
func (application *Application) Run(ctx context.Context) error {
	stageStartedAt := time.Now()
	listener, err := net.Listen("tcp", application.server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", application.server.Addr, err)
	}
	application.logger.Info("[HTTP] listening", "address", listener.Addr().String(), "duration", time.Since(stageStartedAt))

	stageStartedAt = time.Now()
	application.music.Start(ctx)
	application.logger.Info("[WORKER] started", "duration", time.Since(stageStartedAt), "count", application.config.Music.WorkerCount)
	application.ready.Store(true)
	application.logger.Info("[READY] Jiuin Backend is ready")

	err = application.server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (application *Application) Shutdown(ctx context.Context) error {
	var shutdownError error
	application.shutdownOnce.Do(func() {
		application.ready.Store(false)
		application.logger.Info("[SHUTDOWN] stopping HTTP server")
		if err := application.server.Shutdown(ctx); err != nil {
			shutdownError = fmt.Errorf("shutdown HTTP server: %w", err)
		}
		application.logger.Info("[SHUTDOWN] stopping workers")
		application.music.Stop()
		if err := application.closer.Close(); err != nil {
			if shutdownError == nil {
				shutdownError = fmt.Errorf("close music database: %w", err)
			}
			application.logger.Error("[DATABASE] close failed", "error", err)
		}
		application.logger.Info("[SHUTDOWN] complete")
	})
	return shutdownError
}
