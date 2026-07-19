package secret_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func TestDeleteSecret_EmptySecretID(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	sess, _ := openVaultSession(t, "v1")
	_, err := newSecretUC(t, server, sess).DeleteSecret(context.Background(), "v1", "", 1)
	require.ErrorIs(t, err, secret.ErrEmptySecretID)
}

func TestDeleteSecret_VaultLocked(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	_, err := newSecretUC(t, server, session.New()).DeleteSecret(context.Background(), "v1", "s1", 1)
	require.Error(t, err)
}

func TestDeleteSecret_Success(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row"), Version: 1,
	}))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().DeleteSecret(mock.Anything, "tok", "s1", int64(1)).Return(nil)

	conflict, err := newSecretUCStore(t, server, sess, local).DeleteSecret(context.Background(), "v1", "s1", 1)
	require.NoError(t, err)
	assert.Nil(t, conflict)

	_, ok, err := local.GetSecret(context.Background(), "s1")
	require.NoError(t, err)
	assert.False(t, ok, "секрет должен быть удалён из локального кеша")
}

func TestDeleteSecret_Conflict(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		DeleteSecret(mock.Anything, "tok", "s1", int64(1)).
		Return(&grpcclient.ConflictError{Server: serverVersionBlobs(t, vaultKey, "v1", "s1", 5)})

	conflict, err := newSecretUC(t, server, sess).DeleteSecret(context.Background(), "v1", "s1", 1)
	require.NoError(t, err)
	require.NotNil(t, conflict)
	assert.True(t, conflict.IsDelete)
	assert.Equal(t, "s1", conflict.SecretID)
	assert.Equal(t, int64(5), conflict.ServerVersion)
}

func TestDeleteSecret_FallbackOffline(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row"), Version: 1,
	}))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().DeleteSecret(mock.Anything, "tok", "s1", int64(1)).Return(grpcclient.ErrUnavailable)

	conflict, err := newSecretUCStore(t, server, sess, local).DeleteSecret(context.Background(), "v1", "s1", 1)
	require.NoError(t, err)
	assert.Nil(t, conflict)

	entries, err := local.ListPendingOutbox(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "delete должен попасть в outbox при офлайне")
}

func TestDeleteSecret_ServerError(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().DeleteSecret(mock.Anything, "tok", "s1", int64(1)).Return(assert.AnError)

	_, err := newSecretUC(t, server, sess).DeleteSecret(context.Background(), "v1", "s1", 1)
	require.ErrorIs(t, err, assert.AnError)
}
