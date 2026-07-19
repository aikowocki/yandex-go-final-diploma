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
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func TestGetDetail_VaultLocked(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	_, err := newSecretUC(t, server, session.New()).GetDetail(context.Background(), "v1", "s1")
	require.ErrorIs(t, err, secret.ErrVaultLocked)
}

func TestGetDetail_PayloadOnlyWhenNoLocalCache(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}
	encPayload, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierPayload),
		secretcontent.LoginPasswordPayload{V: 1, Password: "hunter2"})
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().GetSecretPayload(mock.Anything, "tok", "s1").Return(contracts.SecretPayloadItem{
		ID: "s1", Type: 1, Version: 1, EncPayload: encPayload,
	}, nil)

	got, err := newSecretUC(t, server, sess).GetDetail(context.Background(), "v1", "s1")
	require.NoError(t, err)
	assert.Equal(t, "s1", got.ID)
	assert.Equal(t, "hunter2", got.Payload.Password)
	assert.Empty(t, got.Row.Title, "без локального кеша Row не заполняется")
}

func TestGetDetail_WithLocalRowAndIndexLoaded(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}
	encPayload, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierPayload),
		secretcontent.LoginPasswordPayload{V: 1, Password: "hunter2"})
	require.NoError(t, err)
	encRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierRow),
		secretcontent.LoginPasswordRow{V: 1, Title: "GitHub", Username: "alice"})
	require.NoError(t, err)
	encIndex, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierIndex),
		secretcontent.LoginPasswordIndex{V: 1, Note: "my-note"})
	require.NoError(t, err)

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: encRow, Version: 1,
	}))
	require.NoError(t, local.SetSecretIndex(context.Background(), "s1", encIndex, 1))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().GetSecretPayload(mock.Anything, "tok", "s1").Return(contracts.SecretPayloadItem{
		ID: "s1", Type: 1, Version: 1, EncPayload: encPayload,
	}, nil)

	got, err := newSecretUCStore(t, server, sess, local).GetDetail(context.Background(), "v1", "s1")
	require.NoError(t, err)
	assert.Equal(t, "GitHub", got.Row.Title)
	assert.Equal(t, "alice", got.Row.Username)
	assert.Equal(t, "my-note", got.Index.Note)
	assert.Equal(t, "hunter2", got.Payload.Password)
}

func TestGetDetail_LoadsIndexFromServerWhenNotYetLoaded(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}
	encPayload, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierPayload),
		secretcontent.LoginPasswordPayload{V: 1, Password: "hunter2"})
	require.NoError(t, err)
	encRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierRow),
		secretcontent.LoginPasswordRow{V: 1, Title: "GitHub"})
	require.NoError(t, err)
	encIndex, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierIndex),
		secretcontent.LoginPasswordIndex{V: 1, Note: "server-note"})
	require.NoError(t, err)

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: encRow, Version: 1,
	}))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().GetSecretPayload(mock.Anything, "tok", "s1").Return(contracts.SecretPayloadItem{
		ID: "s1", Type: 1, Version: 1, EncPayload: encPayload,
	}, nil)
	server.EXPECT().ListSecretIndex(mock.Anything, "tok", "v1").Return([]contracts.SecretIndexItem{
		{ID: "s1", Version: 1, EncIndex: encIndex},
	}, nil)

	got, err := newSecretUCStore(t, server, sess, local).GetDetail(context.Background(), "v1", "s1")
	require.NoError(t, err)
	assert.Equal(t, "server-note", got.Index.Note)
}

func TestGetDetail_PayloadError(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().GetSecretPayload(mock.Anything, "tok", "s1").Return(contracts.SecretPayloadItem{}, assert.AnError)

	_, err := newSecretUC(t, server, sess).GetDetail(context.Background(), "v1", "s1")
	require.ErrorIs(t, err, assert.AnError)
}
