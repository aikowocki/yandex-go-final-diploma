package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClientConfig_Defaults(t *testing.T) {
	cfg, err := parseClientConfig(nil, "/home/user/.config/gophkeeper")
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", cfg.ServerAddr)
	assert.Equal(t, "/home/user/.config/gophkeeper", cfg.DataDir)
	assert.False(t, cfg.NoPersist)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "en", cfg.Lang)
}

func TestParseClientConfig_AllFieldsFromEnv(t *testing.T) {
	t.Setenv("SERVER_ADDR", "remote:5050")
	t.Setenv("DATA_DIR", "/custom/data")
	t.Setenv("NO_PERSIST", "true")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("GOPHKEEPER_LANG", "ru")

	cfg, err := parseClientConfig(nil, "/default/path")
	require.NoError(t, err)

	assert.Equal(t, "remote:5050", cfg.ServerAddr)
	assert.Equal(t, "/custom/data", cfg.DataDir)
	assert.True(t, cfg.NoPersist)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "ru", cfg.Lang)
}

func TestParseClientConfig_Priority(t *testing.T) {
	// flag > env > default
	t.Setenv("SERVER_ADDR", "env-server:4040")
	t.Setenv("DATA_DIR", "/env/data")

	cfg, err := parseClientConfig([]string{"--server-addr", "flag-server:3030"}, "/default/path")
	require.NoError(t, err)

	assert.Equal(t, "flag-server:3030", cfg.ServerAddr, "flag wins over env")
	assert.Equal(t, "/env/data", cfg.DataDir, "env wins over dynamic default")
}

func TestParseClientConfig_JSONConfigFile(t *testing.T) {
	// Создаём temp-директорию с config.json
	dataDir := t.TempDir()
	configContent := `{"server_addr": "from-json:7070", "log_level": "warn"}`
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.json"), []byte(configContent), 0o600))

	cfg, err := parseClientConfig(nil, dataDir)
	require.NoError(t, err)

	assert.Equal(t, "from-json:7070", cfg.ServerAddr, "JSON config overrides default")
	assert.Equal(t, "warn", cfg.LogLevel, "JSON config overrides default")
	assert.Equal(t, dataDir, cfg.DataDir, "DataDir uses dynamic default")
}

func TestParseClientConfig_JSONOverridesEnv(t *testing.T) {
	// kong.Configuration resolver имеет приоритет выше env
	dataDir := t.TempDir()
	configContent := `{"server_addr": "from-json:7070", "log_level": "warn"}`
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.json"), []byte(configContent), 0o600))

	t.Setenv("SERVER_ADDR", "from-env:8080")

	cfg, err := parseClientConfig(nil, dataDir)
	require.NoError(t, err)

	assert.Equal(t, "from-json:7070", cfg.ServerAddr, "JSON config overrides env")
	assert.Equal(t, "warn", cfg.LogLevel, "JSON applies")
}

func TestParseClientConfig_FlagOverridesJSON(t *testing.T) {
	dataDir := t.TempDir()
	configContent := `{"server_addr": "from-json:7070"}`
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.json"), []byte(configContent), 0o600))

	cfg, err := parseClientConfig([]string{"--server-addr", "from-flag:6060"}, dataDir)
	require.NoError(t, err)

	assert.Equal(t, "from-flag:6060", cfg.ServerAddr, "flag overrides JSON config")
}

func TestParseClientConfig_NoConfigFile(t *testing.T) {
	// Директория без config.json — не должно быть ошибки
	dataDir := t.TempDir()

	cfg, err := parseClientConfig(nil, dataDir)
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", cfg.ServerAddr, "defaults work without config file")
}
