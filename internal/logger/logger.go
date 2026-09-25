package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON-formatted slog.Logger.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}
