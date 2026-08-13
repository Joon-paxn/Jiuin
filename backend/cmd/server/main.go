package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Joon-paxn/Jiuin/backend/internal/api"
	"github.com/Joon-paxn/Jiuin/backend/internal/config"
	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	siteRepository := repository.NewConfigSiteRepository(model.SiteInfo{
		Name:    cfg.Site.Name,
		Project: cfg.Site.Project,
		Domain:  cfg.Site.Domain,
	})
	siteService := service.NewSiteService(siteRepository, cfg.Site.Domain)
	musicRepository, err := repository.NewSQLiteMusicRepository(cfg.Music.Directory)
	if err != nil {
		slog.Error("music storage configuration error", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closer, ok := musicRepository.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				logger.Error("music storage close failed", "error", err)
			}
		}
	}()
	musicService := service.NewMusicProcessingService(musicRepository, cfg.Music, logger)
	musicService.Start(context.Background())
	defer musicService.Stop()
	statisticsService := service.NewStatisticsService(repository.NewMemoryStatisticsRepository())
	statusService := service.NewStatusService(repository.NewStaticStatusRepository(model.EcosystemStatus{
		Site: cfg.Ecosystem.MainSiteStatus,
		API:  "online",
		Services: []model.ServiceStatus{
			{Name: "main-site", Status: cfg.Ecosystem.MainSiteStatus},
			{Name: "blog", Status: cfg.Ecosystem.BlogStatus},
			{Name: "api", Status: "online"},
		},
	}))
	links := make([]model.ExternalLink, 0, len(cfg.Ecosystem.ExternalLinks))
	for _, link := range cfg.Ecosystem.ExternalLinks {
		links = append(links, model.ExternalLink{Name: link.Name, URL: link.URL, Description: link.Description})
	}
	linkService := service.NewLinkService(repository.NewStaticLinkRepository(links))
	resources := make([]model.ResourceDescriptor, 0, len(cfg.Ecosystem.Resources))
	for _, resource := range cfg.Ecosystem.Resources {
		resources = append(resources, model.ResourceDescriptor{
			Name: resource.Name, URL: resource.URL, Priority: resource.Priority, CachePolicy: resource.CachePolicy,
		})
	}
	resourceService := service.NewResourceService(repository.NewStaticResourceRepository(resources))

	server := &http.Server{
		Addr: cfg.Server.Address(),
		Handler: api.NewRouter(
			siteService,
			musicService,
			statisticsService,
			statusService,
			linkService,
			resourceService,
			cfg.Ecosystem.SharedServiceToken,
			cfg.Music.AdminToken,
			cfg.CORS.AllowedOrigins,
			logger,
		),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownSignal
		logger.Info("shutdown signal received")

		shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownContext); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	}()

	logger.Info("Jiuin backend started", "environment", cfg.Environment, "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}

	logger.Info("Jiuin backend stopped")
}
