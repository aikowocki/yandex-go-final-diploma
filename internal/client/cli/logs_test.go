package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
)

func TestLogsCmd_Run_NoLogsYet(t *testing.T) {
	cmd := &LogsCmd{Lines: 50}
	require.NoError(t, cmd.Run(&config.ClientConfig{DataDir: t.TempDir()}))
}

func TestLogsCmd_Run_ShowsAllLines(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client.log"), []byte("line1\nline2\n"), 0o600))

	cmd := &LogsCmd{Lines: 0}
	require.NoError(t, cmd.Run(&config.ClientConfig{DataDir: dir}))
}

func TestLogsCmd_Run_ShowsTailLines(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client.log"), []byte("l1\nl2\nl3\nl4\n"), 0o600))

	cmd := &LogsCmd{Lines: 2}
	require.NoError(t, cmd.Run(&config.ClientConfig{DataDir: dir}))
}

func TestLogsCmd_Run_Clear(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "client.log")
	require.NoError(t, os.WriteFile(logPath, []byte("data"), 0o600))

	cmd := &LogsCmd{Clear: true}
	require.NoError(t, cmd.Run(&config.ClientConfig{DataDir: dir}))

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestLogsCmd_Run_ClearNoFile(t *testing.T) {
	cmd := &LogsCmd{Clear: true}
	require.NoError(t, cmd.Run(&config.ClientConfig{DataDir: t.TempDir()}))
}

func TestTailLines(t *testing.T) {
	assert.Equal(t, "l2\nl3\n", tailLines("l1\nl2\nl3\n", 2))
	assert.Equal(t, "l1\nl2\nl3\n", tailLines("l1\nl2\nl3\n", 10))
}
