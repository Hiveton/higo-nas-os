// Command higo-cloud is the HiGoOS cloud control plane + relay. It serves the
// account, device, binding and push REST API for the mobile App and bridges App
// requests to bound NAS devices over the relay tunnel.
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

	"higoos/server-cloud/internal/httpapi"
	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/store"
)

func main() {
	cfg := platform.LoadConfig().WithDefaults()
	logger := platform.NewLogger(cfg.Environment)

	var st store.Store
	switch cfg.Store {
	case "postgres":
		if cfg.DatabaseURL == "" {
			logger.Error("HIGO_CLOUD_STORE=postgres requires HIGO_CLOUD_DB_DSN")
			os.Exit(1)
		}
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		pg, err := store.ConnectPostgres(initCtx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			logger.Error("postgres connect/migrate failed", slog.Any("error", err))
			os.Exit(1)
		}
		defer pg.Close()
		st = pg
		logger.Info("using postgres store")
	default:
		st = store.NewMemory()
		logger.Info("using in-memory store (dev)")
	}

	handler, err := httpapi.NewRouter(httpapi.Dependencies{Config: cfg, Logger: logger, Store: st})
	if err != nil {
		logger.Error("router init failed", slog.Any("error", err))
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("higo-cloud listening", slog.String("addr", cfg.HTTPAddr), slog.String("env", cfg.Environment))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("higo-cloud failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("higo-cloud shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("higo-cloud stopped")
}
