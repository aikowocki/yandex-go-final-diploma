package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func TestTOTPAddCmd_ManualInput(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	// raw secret, issuer, account, title, tags, note
	scriptLines(t, "JBSWY3DPEHPK3PXP", "GitHub", "alice@example.com", "My TOTP", "", "")

	cmd := &TOTPAddCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestTOTPListCmd_Empty(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &TOTPListCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestTOTPCodeCmd_GeneratesCode(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateTOTP(t.Context(), "v1", secretuc.CreateTOTPInput{
		Title: "GitHub", Secret: "JBSWY3DPEHPK3PXP",
	})
	require.NoError(t, err)

	cmd := &TOTPCodeCmd{Vault: "Personal", ID: id}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestPromptTOTPInput_ParsesURI(t *testing.T) {
	scriptLines(t, "otpauth://totp/GitHub:alice?secret=JBSWY3DPEHPK3PXP&issuer=GitHub", "", "", "")
	input, err := promptTOTPInput(testLocalizer())
	require.NoError(t, err)
	assert.Equal(t, "JBSWY3DPEHPK3PXP", input.Secret)
	assert.Equal(t, "GitHub", input.Issuer)
}
