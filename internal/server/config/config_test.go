package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestParseServerConfig_Defaults(t *testing.T) {
	cfg, err := parseServerConfig(nil)
	require.NoError(t, err)

	assert.Equal(t, ":9090", cfg.GRPCAddr)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Empty(t, cfg.DatabaseDSN)
	assert.Empty(t, cfg.JWTSecret)
	assert.Empty(t, cfg.MinioEndpoint)
	assert.Empty(t, cfg.MinioAccess)
	assert.Empty(t, cfg.MinioSecret)
}

func TestParseServerConfig_AllFieldsFromEnv(t *testing.T) {
	t.Setenv("GRPC_ADDR", ":7070")
	t.Setenv("DATABASE_DSN", "postgres://localhost/gophkeeper")
	t.Setenv("JWT_SECRET", "super-secret")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioaccess")
	t.Setenv("MINIO_SECRET_KEY", "miniosecret")

	cfg, err := parseServerConfig(nil)
	require.NoError(t, err)

	assert.Equal(t, ":7070", cfg.GRPCAddr)
	assert.Equal(t, "postgres://localhost/gophkeeper", cfg.DatabaseDSN)
	assert.Equal(t, "super-secret", cfg.JWTSecret)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "localhost:9000", cfg.MinioEndpoint)
	assert.Equal(t, "minioaccess", cfg.MinioAccess)
	assert.Equal(t, "miniosecret", cfg.MinioSecret)
}

func TestParseServerConfig_Priority(t *testing.T) {
	// Проверяем: flag > env > config-file > default
	path := writeTempJSON(t, `{
		"grpc_addr": ":1111",
		"database_dsn": "postgres://json/db",
		"jwt_secret": "json-secret",
		"log_level": "error"
	}`)

	t.Setenv("GRPC_ADDR", ":2222")
	t.Setenv("DATABASE_DSN", "postgres://env/db")

	cfg, err := parseServerConfig([]string{"--config-file", path, "--grpc-addr", ":3333"})
	require.NoError(t, err)

	assert.Equal(t, ":3333", cfg.GRPCAddr, "flag wins over config-file and env")
	assert.Equal(t, "postgres://json/db", cfg.DatabaseDSN, "config-file applied")
	assert.Equal(t, "json-secret", cfg.JWTSecret, "config-file applied")
	assert.Equal(t, "error", cfg.LogLevel, "config-file over default")
}
