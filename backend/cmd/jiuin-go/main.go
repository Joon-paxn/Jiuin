package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/api"
	"github.com/Joon-paxn/Jiuin/backend/internal/core"
	"github.com/Joon-paxn/Jiuin/backend/internal/online"
)

func main() {
	config, err := core.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := config.EnsureStorage(); err != nil {
		log.Fatalf("storage: %v", err)
	}
	db, err := core.OpenDatabase(config.DatabasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	manager := online.New(config.AllowedOrigins)
	server := api.New(config, db, manager)
	workerID := fmt.Sprintf("go-%s-%d", hostname(), os.Getpid())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		err := (core.MusicStore{DB: db, Config: config}).RunWorker(ctx, workerID)
		if err != nil && ctx.Err() == nil {
			log.Printf("backup worker stopped: %v", err)
		}
	}()
	httpServer := &http.Server{Addr: config.ListenAddr, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 2 * time.Minute, IdleTimeout: 60 * time.Second}
	go func() {
		log.Printf("Jiuin Go backup API and WebSocket listening on %s", config.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	manager.Close()
	if err := httpServer.Shutdown(shutdown); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
}

func hostname() string {
	value, err := os.Hostname()
	if err != nil {
		return "host"
	}
	return value
}
