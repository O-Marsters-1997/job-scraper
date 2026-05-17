package utils

import (
	"log/slog"
	"os"
)

func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("required environment variable not set", slog.String("key", key))
		os.Exit(1)
	}
	return value
}
