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
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
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

	// Vault уже отмечен для синхронизации (пользователь выбрал его в scope-попапе).
	// Новые vault'ы по умолчанию SyncEnabled=false и не тянутся, пока не включены.
	require.NoError(t, local.UpsertVault(ctx, contracts.LocalVault{
		ID: "v1", WrappedVaultKey: []byte("w"), EncName: []byte("n"), Version: 1, SyncEnabled: true,
	}))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, "tok").
		Return([]contracts.VaultVersion{{ID: "v1", Version: 2}}, nil)
	server.EXPECT().ListVaults(mock.Anything, "tok").
		Return([]contracts.VaultItem{{ID: "v1", WrappedVaultKey: []byte("w"), EncName: []byte("n"), Version: 2}}, nil).Maybe()
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

// Проигрывание outbox: create-запись отправляется на сервер. id стабилен (client-generated),
// поэтому после успеха просто снимается флаг dirty (без remap временного id).
func TestReplayOutbox_CreateClearsDirty(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)

	// Оффлайн-созданный секрет: строка в кеше (dirty) + запись в outbox с тем же id.
	require.NoError(t, local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row"), EncPayload: []byte("pay"),
		Version: 1, PayloadLoaded: true, Dirty: true,
	}))
	body, err := json.Marshal(contracts.OutboxSecretCreate{
		SecretID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row"), EncPayload: []byte("pay"),
	})
	require.NoError(t, err)
	_, err = local.EnqueueOutbox(ctx, contracts.OutboxEntry{Op: contracts.OutboxOpCreate, Entity: "secret", EntityID: "s1", Payload: body})
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", "s1", "v1", int32(1), []byte("row"), mock.Anything, []byte("pay")).
		Return(nil)

	require.NoError(t, newSyncUC(t, server, local).ReplayOutbox(ctx))

	// Очередь пуста.
	entries, err := local.ListPendingOutbox(ctx)
	require.NoError(t, err)
	assert.Empty(t, entries)

	// Строка сохранила стабильный id и потеряла dirty.
	sec, ok, _ := local.GetSecret(ctx, "s1")
	require.True(t, ok)
	assert.False(t, sec.Dirty)
	assert.True(t, sec.PayloadLoaded)
}

// Проигрывание outbox с конфликтом версий: запись update помечается conflict и НЕ удаляется.
func TestReplayOutbox_UpdateConflictMarksEntry(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)

	require.NoError(t, local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row2"), Version: 2, Dirty: true,
	}))
	body, err := json.Marshal(contracts.OutboxSecretUpdate{
		SecretID: "s1", VaultID: "v1", BaseVersion: 1, Type: 1, EncRow: []byte("row2"), EncIndex: []byte("idx"), EncPayload: []byte("pay"),
	})
	require.NoError(t, err)
	id, err := local.EnqueueOutbox(ctx, contracts.OutboxEntry{Op: contracts.OutboxOpUpdate, Entity: "secret", EntityID: "s1", BaseVersion: 1, Payload: body})
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(1), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(0), &grpcclient.ConflictError{Server: contracts.ServerSecret{ID: "s1", Version: 5}})

	require.NoError(t, newSyncUC(t, server, local).ReplayOutbox(ctx))

	// Запись осталась, но помечена conflict.
	pending, err := local.ListPendingOutbox(ctx)
	require.NoError(t, err)
	assert.Empty(t, pending, "conflict-запись не должна быть pending")

	entry, ok, err := local.GetOutbox(ctx, id)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, contracts.OutboxStatusConflict, entry.Status)
}

// Регрессия: секрет с outbox-записью status=conflict (после неудачного ReplayOutbox — гонка
// версий с другим клиентом) не должен молча затираться серверной версией при следующем Sync.
func TestSync_DoesNotOverwriteSecretWithPendingConflict(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)

	require.NoError(t, local.UpsertVault(ctx, contracts.LocalVault{
		ID: "v1", WrappedVaultKey: []byte("w"), EncName: []byte("n"), Version: 1, SyncEnabled: true,
	}))
	// Локальная (моя) версия секрета — dirty, версия 6 (baseVersion=5 + 1).
	require.NoError(t, local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("mine-row"), Version: 6, Dirty: true,
	}))
	body, err := json.Marshal(contracts.OutboxSecretUpdate{
		SecretID: "s1", VaultID: "v1", BaseVersion: 5, Type: 1, EncRow: []byte("mine-row"),
	})
	require.NoError(t, err)
	_, err = local.EnqueueOutbox(ctx, contracts.OutboxEntry{
		Op: contracts.OutboxOpUpdate, Entity: "secret", EntityID: "s1", BaseVersion: 5,
		Payload: body, Status: contracts.OutboxStatusConflict,
	})
	require.NoError(t, err)

	// Сервер уже применил чужую (другого клиента) версию 7.
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, "tok").
		Return([]contracts.VaultVersion{{ID: "v1", Version: 7}}, nil)
	server.EXPECT().ListSecretRows(mock.Anything, "tok", "v1").
		Return([]contracts.SecretRowItem{{ID: "s1", Type: 1, Version: 7, EncRow: []byte("their-row")}}, nil)

	require.NoError(t, newSyncUC(t, server, local).Sync(ctx))

	// Локальная строка секрета не тронута — версия/enc_row/dirty остались моими.
	sec, ok, err := local.GetSecret(ctx, "s1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(6), sec.Version, "версия не должна перезаписаться, пока конфликт не разрешён")
	assert.Equal(t, []byte("mine-row"), sec.EncRow)
	assert.True(t, sec.Dirty)

	// Синхронизированная версия папки всё же продвинулась (остальные секреты той же папки
	// должны продолжать синхронизироваться нормально).
	v, ok, err := local.GetVault(ctx, "v1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(7), v.SyncedVersion)
}
