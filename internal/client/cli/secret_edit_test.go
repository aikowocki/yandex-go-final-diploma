package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func TestSecretUpdateCmd_NotFoundLocal(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &SecretUpdateCmd{Vault: "Personal", ID: "unknown-id"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretUpdateCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.EXPECT().UpdateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(int64(2), nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateLoginPassword(t.Context(), "v1", loginPasswordInputCLI("GitHub", "alice", "hunter2"))
	require.NoError(t, err)

	scriptLines(t, "GitLab", "bob", "gitlab.com", "", "")
	scriptSecretsCLI(t, "newpass")

	cmd := &SecretUpdateCmd{Vault: "Personal", ID: id}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretDeleteCmd_NotFoundLocal(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &SecretDeleteCmd{Vault: "Personal", ID: "unknown-id", Yes: true}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretDeleteCmd_ConfirmedDelete(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.EXPECT().DeleteSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateLoginPassword(t.Context(), "v1", loginPasswordInputCLI("GitHub", "alice", "hunter2"))
	require.NoError(t, err)

	cmd := &SecretDeleteCmd{Vault: "Personal", ID: id, Yes: true}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretDeleteCmd_DeclinedConfirmation(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateLoginPassword(t.Context(), "v1", loginPasswordInputCLI("GitHub", "alice", "hunter2"))
	require.NoError(t, err)

	scriptLines(t, "n")

	cmd := &SecretDeleteCmd{Vault: "Personal", ID: id}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretSearchCmd_NoMatches(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &SecretSearchCmd{Vault: "Personal", Query: "nothing"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestSecretSearchCmd_WithMatches(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	_, err := env.Secret.CreateLoginPassword(t.Context(), "v1", loginPasswordInputCLI("GitHub", "alice", "hunter2"))
	require.NoError(t, err)

	cmd := &SecretSearchCmd{Vault: "Personal", Query: "GitHub"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestPromptConflictChoice_Mine(t *testing.T) {
	scriptLines(t, "m")
	choice, err := promptConflictChoice(testLocalizer())
	require.NoError(t, err)
	assert.Equal(t, secretuc.ChoiceMine, choice)
}

func TestPromptConflictChoice_Server(t *testing.T) {
	scriptLines(t, "s")
	choice, err := promptConflictChoice(testLocalizer())
	require.NoError(t, err)
	assert.Equal(t, secretuc.ChoiceServer, choice)
}

func TestPromptConflictChoice_InvalidThenValid(t *testing.T) {
	scriptLines(t, "x", "m")
	choice, err := promptConflictChoice(testLocalizer())
	require.NoError(t, err)
	assert.Equal(t, secretuc.ChoiceMine, choice)
}

func TestPromptConflictChoice_AllInvalid(t *testing.T) {
	scriptLines(t, "x", "y", "z")
	_, err := promptConflictChoice(testLocalizer())
	assert.ErrorIs(t, err, errInvalidChoice)
}

func TestPrintGenericConflict_Delete(t *testing.T) {
	printGenericConflict(testLocalizer(), &secretuc.GenericConflict{IsDelete: true})
}

func TestPrintGenericConflict_Update(t *testing.T) {
	printGenericConflict(testLocalizer(), &secretuc.GenericConflict{
		MineRow:   map[string]any{"title": "Mine"},
		ServerRow: map[string]any{"title": "Server"},
	})
}

func TestPrintFieldMap(t *testing.T) {
	printFieldMap(map[string]any{"a": "x", "b": "", "c": nil}) // не должно паниковать
}

func TestLocalTypedVersion(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateLoginPassword(t.Context(), "v1", loginPasswordInputCLI("GitHub", "alice", "hunter2"))
	require.NoError(t, err)

	version, ok, err := localTypedVersion(t.Context(), env.Secret, "v1", id, textSecretType)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(1), version)
}
