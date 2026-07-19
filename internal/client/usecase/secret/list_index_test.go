package secret_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func TestLoadIndexes_EmptyVaultID(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	sess, _ := openVaultSession(t, "v1")
	err := newSecretUC(t, server, sess).LoadIndexes(context.Background(), "")
	require.ErrorIs(t, err, secret.ErrEmptyVaultID)
}

func TestLoadIndexes_CachesForKnownNonDirtySecrets(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row"), Version: 1,
	}))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListSecretIndex(mock.Anything, "tok", "v1").Return([]contracts.SecretIndexItem{
		{ID: "s1", Version: 1, EncIndex: []byte("idx")},
	}, nil)

	err := newSecretUCStore(t, server, sess, local).LoadIndexes(context.Background(), "v1")
	require.NoError(t, err)

	got, ok, err := local.GetSecret(context.Background(), "s1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, got.IndexLoaded)
	assert.Equal(t, []byte("idx"), got.EncIndex)
}

func TestLoadIndexes_SkipsUnknownSecrets(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	local := newMemStore(t)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListSecretIndex(mock.Anything, "tok", "v1").Return([]contracts.SecretIndexItem{
		{ID: "unknown", Version: 1, EncIndex: []byte("idx")},
	}, nil)

	err := newSecretUCStore(t, server, sess, local).LoadIndexes(context.Background(), "v1")
	require.NoError(t, err)

	_, ok, err := local.GetSecret(context.Background(), "unknown")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestLoadIndexes_SkipsDirtySecrets(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row"), Version: 1, Dirty: true,
	}))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListSecretIndex(mock.Anything, "tok", "v1").Return([]contracts.SecretIndexItem{
		{ID: "s1", Version: 1, EncIndex: []byte("idx")},
	}, nil)

	err := newSecretUCStore(t, server, sess, local).LoadIndexes(context.Background(), "v1")
	require.NoError(t, err)

	got, ok, err := local.GetSecret(context.Background(), "s1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.False(t, got.IndexLoaded, "dirty секрет не должен быть затёрт серверным индексом")
}

func TestLoadIndexes_ServerError(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListSecretIndex(mock.Anything, "tok", "v1").Return(nil, assert.AnError)

	err := newSecretUC(t, server, sess).LoadIndexes(context.Background(), "v1")
	require.ErrorIs(t, err, assert.AnError)
}
