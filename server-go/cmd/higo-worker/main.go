package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"higoos/server-go/internal/devstub"
	"higoos/server-go/internal/platform"
)

func main() {
	cfg := platform.LoadConfig()
	logger := platform.NewLogger(cfg.Environment)
	store := devstub.NewStore()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	status := store.Status()
	logger.Info("higo-worker health",
		slog.String("status", status.Status),
		slog.String("adapter", status.Adapter),
		slog.String("env", cfg.Environment),
		slog.Int("app_count", status.AppCount),
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info("higo-worker stopped")
			os.Exit(0)
		case <-ticker.C:
			status := store.Status()
			logger.Info("higo-worker health",
				slog.String("status", status.Status),
				slog.String("adapter", status.Adapter),
				slog.Int("app_count", status.AppCount),
			)
		}
	}
}
