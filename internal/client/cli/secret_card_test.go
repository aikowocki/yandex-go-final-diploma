package cli

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func TestBankCardAddCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	// title, bank, cardholder, brand, expiry, tags, note (после pan/cvv через readSecretFn)
	scriptSecretsCLI(t, "4532015112830366", "123")
	scriptLines(t, "My Card", "Chase", "Alice", "Visa", "12/29", "", "")

	cmd := &BankCardAddCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestBankCardListCmd_Empty(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")

	cmd := &BankCardListCmd{Vault: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestBankCardShowCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env := newCLITestEnv(t, server)
	setupUnlockedVaultCLI(t, env, "Personal")
	id, err := env.Secret.CreateBankCard(t.Context(), "v1", secretuc.CreateBankCardInput{
		Title: "My Card", PAN: "4532015112830366", CVV: "123",
	})
	require.NoError(t, err)

	cmd := &BankCardShowCmd{Vault: "Personal", ID: id}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}
