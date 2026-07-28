package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/pythonsaber2/dockly/internal/deploy"
	"github.com/pythonsaber2/dockly/internal/server"
	"github.com/pythonsaber2/dockly/internal/store"
)

var version = "dev"

func main() {
	listen := flag.String("listen", env("DOCKLY_LISTEN", ":8080"), "HTTP listen address")
	dataDir := flag.String("data-dir", env("DOCKLY_DATA_DIR", "/var/lib/dockly"), "persistent data directory")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s, err := store.Open(filepath.Join(*dataDir, "state.json"))
	if err != nil {
		logger.Error("open state", "error", err)
		os.Exit(1)
	}
	engineReady := deploy.IsDockerAvailable(context.Background())
	if !engineReady {
		logger.Warn("Docker is unavailable; the dashboard will work but deployments will fail until Docker is connected")
	}

	handler := server.New(s, deploy.DefaultEngine(*dataDir, env("DOCKLY_HEALTH_HOST", "127.0.0.1")), server.Config{
		SecureCookies: envBool("DOCKLY_SECURE_COOKIES", false), APIToken: os.Getenv("DOCKLY_API_TOKEN"), Version: version, Logger: logger,
		EngineReady: engineReady,
	})
	httpServer := &http.Server{Addr: *listen, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}

	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		logger.Info("Dockly started", "listen", *listen, "version", version)
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()
	<-stopped
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
