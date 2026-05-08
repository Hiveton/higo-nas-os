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

	"higoos/server-go/internal/httpapi"
	"higoos/server-go/internal/platform"
)

func main() {
	cfg := platform.LoadConfig()
	logger := platform.NewLogger(cfg.Environment)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(httpapi.Dependencies{Config: cfg, Logger: logger}),
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
