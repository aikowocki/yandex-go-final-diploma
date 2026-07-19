package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

func TestFileAddCmd_FileNotFound(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &FileAddCmd{Vault: "Personal", Path: "/nonexistent/file.txt"}
	require.Error(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestFileListCmd_Empty(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &FileListCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestIsDir(t *testing.T) {
	assert.True(t, isDir(t.TempDir()))
	assert.False(t, isDir("/nonexistent/path"))
}
