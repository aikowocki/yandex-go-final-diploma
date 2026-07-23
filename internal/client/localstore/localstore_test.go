package localstore_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
)

func openMem(t *testing.T) *localstore.Store {
	t.Helper()
	ls, err := localstore.Open("", false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ls.Close() })
	return ls
}

func TestKV_SetGet(t *testing.T) {
	ctx := context.Background()
	ls := openMem(t)

	_, ok, err := ls.KVGet(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, ls.KVSet(ctx, "k", []byte("v1")))
	v, ok, err := ls.KVGet(ctx, "k")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("v1"), v)

	require.NoError(t, ls.KVSet(ctx, "k", []byte("v2")))
	v, _, _ = ls.KVGet(ctx, "k")
	assert.Equal(t, []byte("v2"), v)
}

func TestVault_UpsertPreservesSyncedVersion(t *testing.T) {
	ctx := context.Background()
	ls := openMem(t)

	require.NoError(t, ls.UpsertVault(ctx, contracts.LocalVault{
		ID: "v1", WrappedVaultKey: []byte("w"), EncName: []byte("n"), Version: 1,
	}))
	require.NoError(t, ls.SetVaultSyncedVersion(ctx, "v1", 5))

	// Повторный upsert (например, из vault.List) не должен сбрасывать synced_version.
	require.NoError(t, ls.UpsertVault(ctx, contracts.LocalVault{
		ID: "v1", WrappedVaultKey: []byte("w2"), EncName: []byte("n2"), Version: 7,
	}))

	v, ok, err := ls.GetVault(ctx, "v1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(7), v.Version)
	assert.Equal(t, int64(5), v.SyncedVersion)
	assert.Equal(t, []byte("w2"), v.WrappedVaultKey)
}

func TestSecret_UpsertRowKeepsCachedPayload(t *testing.T) {
	ctx := context.Background()
	ls := openMem(t)

	require.NoError(t, ls.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row1"), Version: 1,
	}))
	require.NoError(t, ls.SetSecretPayload(ctx, "s1", []byte("pay"), 1))

	// Повторный upsert с ТОЙ ЖЕ версией не должен затирать закешированный payload.
	require.NoError(t, ls.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row1-bis"), Version: 1,
	}))

	sec, ok, err := ls.GetSecret(ctx, "s1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, []byte("row1-bis"), sec.EncRow)
	assert.Equal(t, int64(1), sec.Version)
	assert.True(t, sec.PayloadLoaded)
	assert.Equal(t, []byte("pay"), sec.EncPayload)
}

// При изменении version (sync подтянул новую версию с сервера) кешированные payload/index
// СБРАСЫВАЮТСЯ — они зашифрованы с AAD, включающим version, и при смене version старый
// enc_payload не расшифруется.
func TestSecret_UpsertRowClearsPayloadOnVersionChange(t *testing.T) {
	ctx := context.Background()
	ls := openMem(t)

	require.NoError(t, ls.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row1"), Version: 1,
	}))
	require.NoError(t, ls.SetSecretPayload(ctx, "s1", []byte("pay"), 1))

	// Upsert с НОВОЙ версией (sync) — payload должен сброситься.
	require.NoError(t, ls.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("row2"), Version: 2,
	}))

	sec, ok, err := ls.GetSecret(ctx, "s1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, []byte("row2"), sec.EncRow)
	assert.Equal(t, int64(2), sec.Version)
	assert.False(t, sec.PayloadLoaded, "payload_loaded must be reset when version changes")
	assert.Nil(t, sec.EncPayload, "enc_payload must be cleared when version changes")
}

func TestSecret_ListByVaultAndDelete(t *testing.T) {
	ctx := context.Background()
	ls := openMem(t)

	require.NoError(t, ls.UpsertSecretRow(ctx, contracts.LocalSecret{ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("a"), Version: 1}))
	require.NoError(t, ls.UpsertSecretRow(ctx, contracts.LocalSecret{ID: "s2", VaultID: "v1", Type: 1, EncRow: []byte("b"), Version: 1}))
	require.NoError(t, ls.UpsertSecretRow(ctx, contracts.LocalSecret{ID: "s3", VaultID: "v2", Type: 1, EncRow: []byte("c"), Version: 1}))

	rows, err := ls.ListSecretsByVault(ctx, "v1")
	require.NoError(t, err)
	assert.Len(t, rows, 2)

	require.NoError(t, ls.DeleteSecret(ctx, "s1"))
	rows, _ = ls.ListSecretsByVault(ctx, "v1")
	assert.Len(t, rows, 1)
}

func TestOutbox_EnqueueListRemove(t *testing.T) {
	ctx := context.Background()
	ls := openMem(t)

	id1, err := ls.EnqueueOutbox(ctx, contracts.OutboxEntry{Op: contracts.OutboxOpCreate, Entity: "secret", EntityID: "t1", Payload: []byte("p1")})
	require.NoError(t, err)
	_, err = ls.EnqueueOutbox(ctx, contracts.OutboxEntry{Op: contracts.OutboxOpCreate, Entity: "secret", EntityID: "t2", Payload: []byte("p2")})
	require.NoError(t, err)

	entries, err := ls.ListPendingOutbox(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "t1", entries[0].EntityID) // FIFO
	assert.NotEmpty(t, entries[0].CreatedAt)

	require.NoError(t, ls.RemoveOutbox(ctx, id1))
	entries, _ = ls.ListPendingOutbox(ctx)
	require.Len(t, entries, 1)
	assert.Equal(t, "t2", entries[0].EntityID)
}
