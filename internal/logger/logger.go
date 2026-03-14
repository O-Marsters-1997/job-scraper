package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON-formatted slog.Logger. Call slog.SetDefault(logger.New())
// once in main; all packages then use slog.Info/Error/Debug directly.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
