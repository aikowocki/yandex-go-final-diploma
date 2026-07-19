package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

// setupUnlockedVault создаёт разблокированную сессию с открытым vault'ом "Personal"/"v1" в
// локальном кеше, через который openVaultByName резолвит имя без сети.
func setupUnlockedVaultCLI(t *testing.T, env *cliTestEnv, vaultName string) string {
	t.Helper()
	env.Session.SetMasterKey(make([]byte, 32))
	env.Server.EXPECT().CreateVault(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("v1", nil).Once()
	id, err := env.Vault.Create(t.Context(), vaultName)
	require.NoError(t, err)
	return id
}

func TestSecretAddCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	scriptLines(t, "GitHub", "alice", "github.com", "", "") // title, username, uri, tags, note
	scriptSecretsCLI(t, "hunter2")                          // password
	// promptOTPCodes читает через readLineFn as well - следующий readLine после note пуст -> stop.
	// (already covered by empty default readLineFn behavior)

	cmd := &SecretAddCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretAddCmd_RequiresUnlock(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{}, assertAnErrorCLI())

	env := newCLITestEnv(t, server)
	cmd := &SecretAddCmd{Vault: "Personal"}
	require.Error(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretListCmd_Empty(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &SecretListCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Sync, env.Localizer))
}

func TestSecretGetCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateLoginPassword(t.Context(), "v1", loginPasswordInputCLI("GitHub", "alice", "hunter2"))
	require.NoError(t, err)

	cmd := &SecretGetCmd{Vault: "Personal", ID: id}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretShowCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.EXPECT().GetSecretPayload(mock.Anything, mock.Anything, mock.Anything).Return(contracts.SecretPayloadItem{}, assertAnErrorCLI()).Maybe()

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateLoginPassword(t.Context(), "v1", loginPasswordInputCLI("GitHub", "alice", "hunter2"))
	require.NoError(t, err)

	cmd := &SecretShowCmd{Vault: "Personal", ID: id}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestParseTags(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, parseTags("a, b"))
	assert.Nil(t, parseTags(""))
	assert.Nil(t, parseTags("  ,  "))
}
