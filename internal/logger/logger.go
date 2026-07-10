package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Setup настраивает глобальный slog логгер с JSON-форматом.
func Setup(level string) {
	SetupWithDir(level, "")
}

// SetupWithDir — расширенная версия Setup: если dataDir непустой, пишет логи в файл
// <dataDir>/client.log (для CLI-клиента). Иначе — в stderr (для сервера/тестов).
func SetupWithDir(level, dataDir string) {
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

	var w io.Writer = os.Stderr
	if dataDir != "" {
		_ = os.MkdirAll(dataDir, 0o700)
		f, err := os.OpenFile(filepath.Join(dataDir, "client.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err == nil {
			w = f
		}
		// Если файл не открылся (read-only fs и т.п.) — fallback на stderr, не паникуем.
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: lvl,
	})
	slog.SetDefault(slog.New(handler))
}
