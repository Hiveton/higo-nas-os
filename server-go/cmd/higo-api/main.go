package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"higoos/server-go/internal/db"
	"higoos/server-go/internal/httpapi"
	"higoos/server-go/internal/platform"
)

func main() {
	cfg := platform.LoadConfig()
	logger := platform.NewLogger(cfg.Environment)

	// Optional Postgres (pgvector) backbone for AI index/search. When no DSN is
	// set the app runs against its JSON-state stubs.
	var pool *db.Pool
	if cfg.DatabaseURL != "" {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if applied, err := db.Migrate(initCtx, cfg.DatabaseURL); err != nil {
			logger.Error("database migration failed", slog.Any("error", err))
		} else {
			logger.Info("database migrations applied", slog.Int("count", applied))
			if p, err := db.Connect(initCtx, cfg.DatabaseURL); err != nil {
				logger.Error("database connect failed", slog.Any("error", err))
			} else {
				pool = p
				logger.Info("database connected")
			}
		}
		cancel()
	}
	if pool != nil {
		defer pool.Close()
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(httpapi.Dependencies{Config: cfg, Logger: logger, DB: pool}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("higo-api listening", slog.String("addr", cfg.HTTPAddr), slog.String("env", cfg.Environment))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("higo-api failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("higo-api shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("higo-api stopped")
}
