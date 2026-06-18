package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/tasks"
)

// higo-worker is a lightweight sidecar that reports on the background task
// ledger written by higo-api. Task execution itself is hosted in-process by
// higo-api (see internal/tasks): a JSON-file queue cannot be safely driven by a
// second writer, so cross-process consumption waits for the database phase.
// Until then the worker observes real queue health read-only instead of the old
// devstub heartbeat.
func main() {
	cfg := platform.LoadConfig()
	logger := platform.NewLogger(cfg.Environment)

	ledgerPath := ""
	if cfg.StateDir != "" {
		ledgerPath = filepath.Join(cfg.StateDir, "tasks.json")
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	report := func() {
		list, err := tasks.ReadLedger(ledgerPath)
		if err != nil {
			logger.Warn("higo-worker ledger read failed", slog.Any("error", err))
			return
		}
		stats := tasks.Summarize(list)
		logger.Info("higo-worker health",
			slog.String("env", cfg.Environment),
			slog.Int("tasks_total", stats.Total),
			slog.Int("tasks_queued", stats.Queued),
			slog.Int("tasks_running", stats.Running),
			slog.Int("tasks_failed", stats.Failed),
		)
	}
	report()

	for {
		select {
		case <-ctx.Done():
			logger.Info("higo-worker stopped")
			os.Exit(0)
		case <-ticker.C:
			report()
		}
	}
}
