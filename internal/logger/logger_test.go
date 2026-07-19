package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetup_DefaultsToStderr(t *testing.T) {
	Setup("info")
	assert.Equal(t, slog.LevelInfo, currentLevel())
}

func TestSetupWithDir_LevelsParsed(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			SetupWithDir(tt.in, "")
			assert.Equal(t, tt.want, currentLevel())
		})
	}
}

func TestSetupWithDir_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	SetupWithDir("info", dir)

	slog.Info("hello from test")

	data, err := os.ReadFile(filepath.Join(dir, "client.log"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "hello from test")

	// Возвращаем логгер в состояние по умолчанию, чтобы не влиять на другие тесты пакета.
	Setup("info")
}

func TestSetupWithDir_FallsBackToStderrOnBadDir(t *testing.T) {
	// Путь не может быть создан как директория (файл существует под тем же именем).
	base := t.TempDir()
	blocked := filepath.Join(base, "blocked")
	require.NoError(t, os.WriteFile(blocked, []byte("x"), 0o600))

	// Не должно паниковать — тихий fallback на stderr.
	assert.NotPanics(t, func() {
		SetupWithDir("info", filepath.Join(blocked, "sub"))
	})

	Setup("info")
}

// currentLevel извлекает уровень текущего дефолтного логгера через Enabled-проверки.
func currentLevel() slog.Level {
	h := slog.Default().Handler()
	for _, lvl := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		if h.Enabled(context.Background(), lvl) {
			return lvl
		}
	}
	return slog.LevelError
}
