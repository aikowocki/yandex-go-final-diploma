package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

func TestVaultCreateCmd_RequiresUnlock(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{}, assert.AnError)

	env := newCLITestEnv(t, server)
	cmd := &VaultCreateCmd{Name: "Personal"}
	err := cmd.Run(env.Auth, env.Vault, env.Localizer)
	require.Error(t, err)
}

func TestVaultCreateCmd_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateVault(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("v1", nil)

	env := newCLITestEnv(t, server)
	env.Session.SetMasterKey(make([]byte, 32))

	cmd := &VaultCreateCmd{Name: "Personal"}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Localizer))
}

func TestVaultListCmd_Empty(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil)

	env := newCLITestEnv(t, server)
	env.Session.SetMasterKey(make([]byte, 32))

	cmd := &VaultListCmd{}
	require.NoError(t, cmd.Run(env.Auth, env.Vault, env.Localizer))
}

func TestVaultListCmd_RequiresUnlock(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{}, assert.AnError)

	env := newCLITestEnv(t, server)
	cmd := &VaultListCmd{}
	err := cmd.Run(env.Auth, env.Vault, env.Localizer)
	require.Error(t, err)
}

func TestResolveVaultID_NotFound(t *testing.T) {
	_, err := resolveVaultID(nil, "Personal")
	assert.ErrorIs(t, err, errVaultNotFound)
}

func TestResolveVaultID_Ambiguous(t *testing.T) {
	vaults := []vaultuc.DecryptedVault{{ID: "v1", Name: "Personal"}, {ID: "v2", Name: "Personal"}}
	_, err := resolveVaultID(vaults, "Personal")
	assert.ErrorIs(t, err, errVaultAmbiguous)
}

func TestResolveVaultID_Success(t *testing.T) {
	vaults := []vaultuc.DecryptedVault{{ID: "v1", Name: "Personal"}, {ID: "v2", Name: "Work"}}
	id, err := resolveVaultID(vaults, "Work")
	require.NoError(t, err)
	assert.Equal(t, "v2", id)
}

func TestOpenVaultByName_FallsBackToServer(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	env := newCLITestEnv(t, server)
	env.Session.SetMasterKey(make([]byte, 32))

	_, err := openVaultByName(context.Background(), env.Vault, "Unknown")
	require.Error(t, err)
}
