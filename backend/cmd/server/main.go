package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/app"
	"github.com/Joon-paxn/Jiuin/backend/internal/config"
)

func main() {
	startedAt := time.Now()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("[CONFIG] failed", "error", err)
		os.Exit(1)
	}
	// Configuration validation has already resolved this location. Set it once
	// for process logs and any local-time presentation; persisted timestamps
	// continue to use UTC in repositories and services.
	location, _ := time.LoadLocation(cfg.Timezone)
	time.Local = location
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: config.LogLevel(cfg.LogLevel)}))
	logger.Info("[CONFIG] ready", "duration", time.Since(startedAt), "timezone", cfg.Timezone)

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("[BOOT] failed", "error", err)
		os.Exit(1)
	}

	rootContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	runResult := make(chan error, 1)
	go func() { runResult <- application.Run(rootContext) }()

	select {
	case err := <-runResult:
		if err != nil {
			logger.Error("[HTTP] stopped unexpectedly", "error", err)
			_ = application.Shutdown(context.Background())
			os.Exit(1)
		}
	case <-rootContext.Done():
		logger.Info("[SHUTDOWN] signal received")
		shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err := application.Shutdown(shutdownContext); err != nil {
			logger.Error("[SHUTDOWN] failed", "error", err)
			os.Exit(1)
		}
		if err := <-runResult; err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("[HTTP] stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}
	logger.Info("Jiuin backend stopped")
}
