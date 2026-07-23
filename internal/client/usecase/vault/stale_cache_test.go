package vault_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
)

// seedRawVault кладёт в кеш папку с произвольными (заведомо неразворачиваемыми текущим
// MasterKey) шифротекстами — имитирует "осиротевшую" запись от другого аккаунта/пароля.
func seedRawVault(t *testing.T, local *localstore.Store, id string) {
	t.Helper()
	require.NoError(t, local.UpsertVault(context.Background(), contracts.LocalVault{
		ID:              id,
		WrappedVaultKey: []byte("not-a-real-wrapped-key-but-24-bytes-plus"),
		EncName:         []byte("garbage"),
		Version:         1,
	}))
}

// TestListLocal_SkipsUndecryptableEntries: ListLocal не должен обрушиваться из-за одной
// "осиротевшей" записи (другой MasterKey/повреждённые данные) — остальные папки должны
// остаться доступны.
func TestListLocal_SkipsUndecryptableEntries(t *testing.T) {
	sess := unlockedSession(t)
	mk, _ := sess.MasterKey()
	local := newMemStore(t)

	// Валидная папка, разворачиваемая текущим MasterKey.
	c := cryptoimpl.Crypto{}
	vaultKey, err := c.GenerateVaultKey()
	require.NoError(t, err)
	wrapped, err := c.WrapVaultKey(vaultKey, mk)
	require.NoError(t, err)
	encName, err := c.EncryptStruct(vaultKey, nil, "Good Vault")
	require.NoError(t, err)
	require.NoError(t, local.UpsertVault(context.Background(), contracts.LocalVault{
		ID: "good-1", WrappedVaultKey: wrapped, EncName: encName, Version: 1,
	}))

	// "Осиротевшая" запись — не разворачивается текущим MasterKey.
	seedRawVault(t, local, "stale-1")

	server := mocks.NewMockServerClient(t) // ListLocal не должен ходить в сеть
	sess2 := sess
	uc := newVaultUCStore(t, server, sess2, local)

	got, err := uc.ListLocal(context.Background())
	require.NoError(t, err, "one undecryptable entry must not fail the whole call")
	require.Len(t, got, 1)
	assert.Equal(t, "Good Vault", got[0].Name)
}

// TestList_PrunesStaleVaultsNotReturnedByServer: List() удаляет из локального кеша папки,
// которых сервер не вернул — иначе "осиротевшие" записи (от предыдущего аккаунта на том же
// --data-dir, либо реально удалённые папки) остаются в кеше навсегда.
func TestList_PrunesStaleVaultsNotReturnedByServer(t *testing.T) {
	sess := unlockedSession(t)
	mk, _ := sess.MasterKey()
	local := newMemStore(t)

	seedRawVault(t, local, "stale-1")

	c := cryptoimpl.Crypto{}
	vaultKey, err := c.GenerateVaultKey()
	require.NoError(t, err)
	wrapped, err := c.WrapVaultKey(vaultKey, mk)
	require.NoError(t, err)
	encName, err := c.EncryptStruct(vaultKey, nil, "Current")
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListVaults(mock.Anything, "tok").Return([]contracts.VaultItem{
		{ID: "current-1", WrappedVaultKey: wrapped, EncName: encName, Version: 1},
	}, nil)

	uc := newVaultUCStore(t, server, sess, local)
	got, err := uc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "current-1", got[0].ID)

	// Кеш очищен от stale-1 — ListLocal (offline path) больше не видит его и не пытается развернуть.
	cached, err := local.ListVaults(context.Background())
	require.NoError(t, err)
	require.Len(t, cached, 1)
	assert.Equal(t, "current-1", cached[0].ID)
}
