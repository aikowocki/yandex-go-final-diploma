package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup настраивает глобальный slog логгер с JSON-форматом.
func Setup(level string) {
	var lvl slog.Level

	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	slog.SetDefault(slog.New(handler))
}
