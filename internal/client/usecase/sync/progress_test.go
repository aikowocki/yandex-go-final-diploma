package sync_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
)

func TestSyncProgress_String(t *testing.T) {
	assert.Equal(t, "↻ Загрузка папок...", syncuc.Progress{Stage: "vaults"}.String())
	assert.Equal(t, "↻ Синхронизация папок (1/3)", syncuc.Progress{Stage: "rows", Done: 1, Total: 3}.String())
	assert.Equal(t, "↻ Расширенная синхронизация (2/5)", syncuc.Progress{Stage: "index", Done: 2, Total: 5}.String())
	assert.Equal(t, "↻ Синхронизация...", syncuc.Progress{Stage: "unknown"}.String())
}

func TestSyncWithProgress_EmitsStagesAndClosesChannel(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)
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

	uc := newSyncUC(t, server, local)
	ch := make(chan syncuc.Progress, 10)

	err := uc.SyncWithProgress(ctx, ch)
	require.NoError(t, err)

	var stages []string
	for p := range ch {
		stages = append(stages, p.Stage)
	}
	assert.Contains(t, stages, "vaults")
	assert.Contains(t, stages, "rows")
}

func TestSyncWithProgress_ChecksFreshnessError(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, "tok").Return(nil, assert.AnError)

	uc := newSyncUC(t, server, local)
	ch := make(chan syncuc.Progress, 10)

	err := uc.SyncWithProgress(ctx, ch)
	require.ErrorIs(t, err, assert.AnError)

	// Канал должен быть закрыт даже при ошибке (дренируем уже отправленные стадии).
	for range ch {
	}
	_, ok := <-ch
	assert.False(t, ok)
}

func TestSyncWithProgress_LoadsIndexesWhenLoaderSet(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)
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

	uc := newSyncUC(t, server, local)
	loader := &fakeIndexLoader{}
	uc.SetIndexLoader(loader)

	ch := make(chan syncuc.Progress, 10)
	require.NoError(t, uc.SyncWithProgress(ctx, ch))

	var stages []string
	for p := range ch {
		stages = append(stages, p.Stage)
	}
	assert.Contains(t, stages, "index")
	assert.Equal(t, []string{"v1"}, loader.calledVaultIDs)
}

func TestSyncWithProgress_IndexLoaderErrorIsIgnored(t *testing.T) {
	ctx := context.Background()
	local := openMem(t)
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

	uc := newSyncUC(t, server, local)
	uc.SetIndexLoader(&fakeIndexLoader{err: assert.AnError})

	ch := make(chan syncuc.Progress, 10)
	// Ошибка загрузки индекса не должна фейлить весь sync.
	require.NoError(t, uc.SyncWithProgress(ctx, ch))
	for range ch {
	}
}

type fakeIndexLoader struct {
	err            error
	calledVaultIDs []string
}

func (f *fakeIndexLoader) LoadIndexes(ctx context.Context, vaultID string) error {
	f.calledVaultIDs = append(f.calledVaultIDs, vaultID)
	return f.err
}
