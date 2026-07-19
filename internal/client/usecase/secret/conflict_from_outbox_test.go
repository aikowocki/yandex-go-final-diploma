package secret_test

import (
	"context"
	"encoding/json"
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

// cryptoRow шифрует минимальный LoginPasswordRow (только title) под заданной версией — для
// сборки outbox-payload "моей" версии в тестах конфликтов.
func cryptoRow(vaultKey []byte, vaultID, secretID string, version int64, title string) ([]byte, error) {
	c := cryptoimpl.Crypto{}
	return c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, secretID, version, secret.TierRow),
		secretcontent.LoginPasswordRow{V: 1, Title: title})
}

// enqueueConflictUpdate кладёт в outbox update-запись со статусом conflict — как это делает
// ReplayOutbox после handleReplayConflict.
func enqueueConflictUpdate(t *testing.T, local interface {
	EnqueueOutbox(ctx context.Context, e contracts.OutboxEntry) (int64, error)
}, secretID, vaultID string, baseVersion int64, encRow []byte) int64 {
	t.Helper()
	body, err := json.Marshal(contracts.OutboxSecretUpdate{
		SecretID: secretID, VaultID: vaultID, BaseVersion: baseVersion, Type: 1, EncRow: encRow,
	})
	require.NoError(t, err)
	id, err := local.EnqueueOutbox(context.Background(), contracts.OutboxEntry{
		Op: contracts.OutboxOpUpdate, Entity: "secret", EntityID: secretID, BaseVersion: baseVersion,
		Payload: body, Status: contracts.OutboxStatusConflict,
	})
	require.NoError(t, err)
	return id
}

func TestListOutboxConflicts_ReturnsOnlySecretConflicts(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	local := newMemStore(t)
	server := mocks.NewMockServerClient(t)

	mineRow, err := cryptoRow(vaultKey, "v1", "s1", 6, "mine-title")
	require.NoError(t, err)
	enqueueConflictUpdate(t, local, "s1", "v1", 5, mineRow)

	uc := newSecretUCStore(t, server, sess, local)
	conflicts, err := uc.ListOutboxConflicts(context.Background())
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, "s1", conflicts[0].EntityID)
	assert.Equal(t, contracts.OutboxStatusConflict, conflicts[0].Status)
}

func TestConflictFromOutbox_UpdateBuildsConflict(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	local := newMemStore(t)
	server := mocks.NewMockServerClient(t)

	mineRow, err := cryptoRow(vaultKey, "v1", "s1", 6, "mine-title")
	require.NoError(t, err)
	entryID := enqueueConflictUpdate(t, local, "s1", "v1", 5, mineRow)

	serverSecret := serverVersionBlobs(t, vaultKey, "v1", "s1", 7)
	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(5), mineRow, mock.Anything, mock.Anything).
		Return(int64(0), &grpcclient.ConflictError{Server: serverSecret})

	uc := newSecretUCStore(t, server, sess, local)
	conflict, err := uc.ConflictFromOutbox(context.Background(), entryID)
	require.NoError(t, err)
	require.NotNil(t, conflict)

	assert.Equal(t, "s1", conflict.SecretID)
	assert.Equal(t, "mine-title", conflict.MineRow["title"])
	assert.Equal(t, "ServerTitle", conflict.ServerRow["title"])
	assert.Equal(t, int64(7), conflict.ServerVersion)
}

func TestConflictFromOutbox_AlreadyResolvedRemovesEntry(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	local := newMemStore(t)
	server := mocks.NewMockServerClient(t)

	mineRow, err := cryptoRow(vaultKey, "v1", "s1", 6, "mine-title")
	require.NoError(t, err)
	entryID := enqueueConflictUpdate(t, local, "s1", "v1", 5, mineRow)

	// Сервер внезапно принимает запись (гонка разрешилась сама, например пользователь уже
	// вручную сохранил именно это же содержимое с другого места).
	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(5), mineRow, mock.Anything, mock.Anything).
		Return(int64(6), nil)

	uc := newSecretUCStore(t, server, sess, local)
	conflict, err := uc.ConflictFromOutbox(context.Background(), entryID)
	require.NoError(t, err)
	assert.Nil(t, conflict)

	_, ok, err := local.GetOutbox(context.Background(), entryID)
	require.NoError(t, err)
	assert.False(t, ok, "запись должна быть удалена из outbox после успешного повтора")
}

func TestConflictFromOutbox_UnknownEntry(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	local := newMemStore(t)
	server := mocks.NewMockServerClient(t)

	uc := newSecretUCStore(t, server, sess, local)
	_, err := uc.ConflictFromOutbox(context.Background(), 999)
	require.ErrorIs(t, err, secret.ErrOutboxEntryNotFound)
}
