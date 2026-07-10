package sync_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
)

func newSyncUC(t *testing.T, server contracts.ServerClient, local *localstore.Store) *syncuc.UseCase {
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{AccessToken: "tok"}, nil).Maybe()
	return syncuc.New(server, local, store)
}

func openMem(t *testing.T) *localstore.Store {
	t.Helper()
	ls, err := localstore.Open("", false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ls.Close() })
	return ls
}

// Первый синк: кеш пуст → тянет метаданные папки (ListVaults) и строки (ListSecretRows),
// проставляет synced_version.
func TestSync_FreshPull(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, "tok").
		Return([]contracts.VaultVersion{{ID: "v1", Version: 2}}, nil)
	server.EXPECT().ListVaults(mock.Anything, "tok").
		Return([]contracts.VaultItem{{ID: "v1", WrappedVaultKey: []byte("w"), EncName: []byte("n"), Version: 2}}, nil)
	server.EXPECT().ListSecretRows(mock.Anything, "tok", "v1").
		Return([]contracts.SecretRowItem{{ID: "s1", Type: 1, Version: 2, EncRow: []byte("row")}}, nil)

	require.NoError(t, newSyncUC(t, server, local).Sync(ctx))

	v, ok, err := local.GetVault(ctx, "v1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(2), v.SyncedVersion)

	rows, err := local.ListSecretsByVault(ctx, "v1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, []byte("row"), rows[0].EncRow)
}

// Если версия папки не изменилась с последнего синка — ListRow повторно не вызывается.
func TestSync_SkipsUnchangedVault(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)
	require.NoError(t, local.UpsertVault(ctx, contracts.LocalVault{ID: "v1", WrappedVaultKey: []byte("w"), EncName: []byte("n"), Version: 3}))
	require.NoError(t, local.SetVaultSyncedVersion(ctx, "v1", 3))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, "tok").
		Return([]contracts.VaultVersion{{ID: "v1", Version: 3}}, nil)
	// ListVaults/ListSecretRows НЕ ожидаются — mockery провалит тест, если они будут вызваны.

	require.NoError(t, newSyncUC(t, server, local).Sync(ctx))
}

// Проигрывание outbox: create-запись отправляется на сервер, temp-id заменяется на серверный.
func TestReplayOutbox_CreateReconciles(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)

	// Оффлайн-созданный секрет: временная строка в кеше + запись в outbox.
	require.NoError(t, local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "temp-1", VaultID: "v1", Type: 1, EncRow: []byte("row"), Version: 1, Dirty: true,
	}))
	body, err := json.Marshal(contracts.OutboxSecretCreate{
		VaultID: "v1", TempID: "temp-1", Type: 1, EncRow: []byte("row"), EncPayload: []byte("pay"),
	})
	require.NoError(t, err)
	_, err = local.EnqueueOutbox(ctx, contracts.OutboxEntry{Op: contracts.OutboxOpCreate, Entity: "secret", EntityID: "temp-1", Payload: body})
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", "v1", int32(1), []byte("row"), mock.Anything, []byte("pay")).
		Return("server-1", nil)

	require.NoError(t, newSyncUC(t, server, local).ReplayOutbox(ctx))

	// Очередь пуста.
	entries, err := local.ListPendingOutbox(ctx)
	require.NoError(t, err)
	assert.Empty(t, entries)

	// temp-строка удалена, появилась строка с серверным id.
	_, ok, _ := local.GetSecret(ctx, "temp-1")
	assert.False(t, ok)
	sec, ok, _ := local.GetSecret(ctx, "server-1")
	require.True(t, ok)
	assert.False(t, sec.Dirty)
	assert.True(t, sec.PayloadLoaded)
}
