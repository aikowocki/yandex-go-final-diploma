package secret_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// serverVersionBlobs шифрует серверную версию секрета (все три тира) под vaultKey с AAD её версии.
func serverVersionBlobs(t *testing.T, vaultKey []byte, vaultID, secretID string, version int64) contracts.ServerSecret {
	t.Helper()
	c := cryptoimpl.Crypto{}
	encRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, secretID, version, secret.TierRow),
		secretcontent.LoginPasswordRow{V: 1, Title: "ServerTitle", Username: "srv"})
	require.NoError(t, err)
	encIndex, err := c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, secretID, version, secret.TierIndex),
		secretcontent.LoginPasswordIndex{V: 1, Note: "server-note"})
	require.NoError(t, err)
	encPayload, err := c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, secretID, version, secret.TierPayload),
		secretcontent.LoginPasswordPayload{V: 1, Password: "srvpass"})
	require.NoError(t, err)
	return contracts.ServerSecret{ID: secretID, Type: 1, Version: version, EncRow: encRow, EncIndex: encIndex, EncPayload: encPayload}
}

func TestUpdate_ConflictDecryptsBothVersions(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(3), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(0), &grpcclient.ConflictError{Server: serverVersionBlobs(t, vaultKey, "v1", "s1", 5)})

	uc := newSecretUC(t, server, sess)
	conflict, err := uc.UpdateLoginPassword(context.Background(), "v1", "s1", 3, secret.CreateLoginPasswordInput{
		Title: "MyTitle", Username: "me", Password: "mypass",
	})
	require.NoError(t, err)
	require.NotNil(t, conflict)

	assert.Equal(t, "MyTitle", conflict.MineRow["title"])
	assert.Equal(t, "ServerTitle", conflict.ServerRow["title"])
	assert.Equal(t, "server-note", conflict.ServerIndex["note"])
	assert.Equal(t, "srvpass", conflict.ServerPayload["password"])
	assert.Equal(t, int64(5), conflict.ServerVersion)
}

// ChoiceMine: повторный Update с base=server.Version перезатирает серверную версию.
func TestResolveConflict_Mine(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	local := newMemStore(t)
	server := mocks.NewMockServerClient(t)

	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(3), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(0), &grpcclient.ConflictError{Server: serverVersionBlobs(t, vaultKey, "v1", "s1", 5)}).Once()
	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(5), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(6), nil).Once()

	uc := newSecretUCStore(t, server, sess, local)
	conflict, err := uc.UpdateLoginPassword(context.Background(), "v1", "s1", 3, secret.CreateLoginPasswordInput{
		Title: "MyTitle", Username: "me", Password: "mypass",
	})
	require.NoError(t, err)
	require.NotNil(t, conflict)

	next, err := uc.GenericResolveConflict(context.Background(), conflict, secret.ChoiceMine)
	require.NoError(t, err)
	assert.Nil(t, next, "перезапись прошла без нового конфликта")

	sec, ok, err := local.GetSecret(context.Background(), "s1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(6), sec.Version)
	assert.False(t, sec.Dirty)
}

// ChoiceServer: серверная версия принимается, локальные изменения отбрасываются.
func TestResolveConflict_Server(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	local := newMemStore(t)
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(3), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(0), &grpcclient.ConflictError{Server: serverVersionBlobs(t, vaultKey, "v1", "s1", 5)}).Once()

	uc := newSecretUCStore(t, server, sess, local)
	conflict, err := uc.UpdateLoginPassword(context.Background(), "v1", "s1", 3, secret.CreateLoginPasswordInput{
		Title: "MyTitle", Username: "me", Password: "mypass",
	})
	require.NoError(t, err)
	require.NotNil(t, conflict)

	next, err := uc.GenericResolveConflict(context.Background(), conflict, secret.ChoiceServer)
	require.NoError(t, err)
	assert.Nil(t, next)

	// В кеше — серверная версия (5), не dirty; расшифровка через ListRow даёт серверный title.
	sec, ok, err := local.GetSecret(context.Background(), "s1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(5), sec.Version)
	assert.False(t, sec.Dirty)

	rows, err := uc.ListRow(context.Background(), "v1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "ServerTitle", rows[0].Row.Title)
}
