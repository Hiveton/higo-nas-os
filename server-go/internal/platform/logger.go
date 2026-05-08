package platform

import (
	"log/slog"
	"os"
)

func NewLogger(environment string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if environment == "dev" || environment == "test" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
