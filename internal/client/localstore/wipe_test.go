package localstore_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// TestWipeAccountData_ClearsAllTables проверяет, что WipeAccountData стирает vaults/secrets/
// outbox/kv целиком.
func TestWipeAccountData_ClearsAllTables(t *testing.T) {
	ctx := context.Background()
	ls := openMem(t)

	require.NoError(t, ls.UpsertVault(ctx, contracts.LocalVault{
		ID: "v1", WrappedVaultKey: []byte("w"), EncName: []byte("n"), Version: 1,
	}))
	require.NoError(t, ls.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 1, EncRow: []byte("r"), Version: 1,
	}))
	_, err := ls.EnqueueOutbox(ctx, contracts.OutboxEntry{
		Op: contracts.OutboxOpCreate, Entity: "secret", EntityID: "s1", Payload: []byte("{}"),
	})
	require.NoError(t, err)
	require.NoError(t, ls.KVSet(ctx, "auth.account_user_id", []byte("user-1")))

	require.NoError(t, ls.WipeAccountData(ctx))

	vaults, err := ls.ListVaults(ctx)
	require.NoError(t, err)
	assert.Empty(t, vaults)

	secrets, err := ls.ListSecretsByVault(ctx, "v1")
	require.NoError(t, err)
	assert.Empty(t, secrets)

	outbox, err := ls.ListPendingOutbox(ctx)
	require.NoError(t, err)
	assert.Empty(t, outbox)

	_, ok, err := ls.KVGet(ctx, "auth.account_user_id")
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestWipeAccountData_Idempotent: повторный вызов на пустой БД не должен возвращать ошибку.
func TestWipeAccountData_Idempotent(t *testing.T) {
	ls := openMem(t)
	require.NoError(t, ls.WipeAccountData(context.Background()))
	require.NoError(t, ls.WipeAccountData(context.Background()))
}
