package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func TestTextAddCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	scriptLines(t, "My Note", "note body", "", "") // title, body, tags, note

	cmd := &TextAddCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestTextListCmd_Empty(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &TextListCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestTextShowCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateText(t.Context(), "v1", secretuc.CreateTextInput{Title: "Note", Body: "body"})
	require.NoError(t, err)

	cmd := &TextShowCmd{Vault: "Personal", ID: id}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestJoinTags(t *testing.T) {
	assert.Equal(t, "", joinTags(nil))
	assert.Equal(t, "a", joinTags([]string{"a"}))
	assert.Equal(t, "a,b", joinTags([]string{"a", "b"}))
}
