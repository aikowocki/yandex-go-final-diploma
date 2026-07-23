package auth_test

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
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// TestLogin_SameAccount_KeepsCache: повторный Login тем же user_id (обычный сценарий —
// повторный вход/несколько устройств одного аккаунта) НЕ должен трогать локальный кеш.
func TestLogin_SameAccount_KeepsCache(t *testing.T) {
	local := memStore()
	seedVault(t, local, "vault-1")

	res := contracts.LoginResult{Tokens: contracts.Tokens{AccessToken: "a", RefreshToken: "r", UserID: "user-1"}}
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(res, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(res.Tokens).Return(nil)

	uc := authuc.New(server, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, store, session.New(), local)
	require.NoError(t, uc.Login(context.Background(), "alice", []byte("pw")))

	vaults, err := local.ListVaults(context.Background())
	require.NoError(t, err)
	assert.Len(t, vaults, 1, "cache for the same account must be preserved")
}

// TestLogin_DifferentAccount_WipesCache: если userID отличается от того, чьи данные лежат
// в кеше (--data-dir пережил смену аккаунта на сервере), весь локальный кеш должен быть
// стёрт перед продолжением.
func TestLogin_DifferentAccount_WipesCache(t *testing.T) {
	local := memStore()
	seedVault(t, local, "vault-1")

	// Первый логин — аккаунт user-1, кеш помечается им.
	firstRes := contracts.LoginResult{Tokens: contracts.Tokens{AccessToken: "a1", RefreshToken: "r1", UserID: "user-1"}}
	server1 := mocks.NewMockServerClient(t)
	server1.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(firstRes, nil)
	server1.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	store1 := mocks.NewMockTokenStore(t)
	store1.EXPECT().Save(firstRes.Tokens).Return(nil)
	uc1 := authuc.New(server1, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, store1, session.New(), local)
	require.NoError(t, uc1.Login(context.Background(), "alice", []byte("pw")))

	vaults, err := local.ListVaults(context.Background())
	require.NoError(t, err)
	require.Len(t, vaults, 1, "sanity check: cache seeded before account switch")

	// Второй логин — ДРУГОЙ user_id (тот же --data-dir, новый/пересозданный аккаунт на сервере).
	secondRes := contracts.LoginResult{Tokens: contracts.Tokens{AccessToken: "a2", RefreshToken: "r2", UserID: "user-2"}}
	server2 := mocks.NewMockServerClient(t)
	server2.EXPECT().Login(mock.Anything, "bob", []byte("pw2")).Return(secondRes, nil)
	server2.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	store2 := mocks.NewMockTokenStore(t)
	store2.EXPECT().Save(secondRes.Tokens).Return(nil)
	uc2 := authuc.New(server2, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, store2, session.New(), local)
	require.NoError(t, uc2.Login(context.Background(), "bob", []byte("pw2")))

	vaultsAfter, err := local.ListVaults(context.Background())
	require.NoError(t, err)
	assert.Empty(t, vaultsAfter, "cache must be wiped on account switch")
}

// TestRegister_DifferentAccount_WipesCache: тот же сценарий, но через Register (например
// --data-dir использовался ранее для другого аккаунта, который был удалён и создан заново).
func TestRegister_DifferentAccount_WipesCache(t *testing.T) {
	local := memStore()
	seedVault(t, local, "vault-old")

	require.NoError(t, local.KVSet(context.Background(), "auth.account_user_id", []byte("user-old")))

	tokens := contracts.Tokens{AccessToken: "a", RefreshToken: "r", UserID: "user-new"}
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Register(mock.Anything, "carol", []byte("pw")).Return(tokens, nil)
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(tokens).Return(nil)

	uc := authuc.New(server, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, store, session.New(), local)
	require.NoError(t, uc.Register(context.Background(), "carol", []byte("pw")))

	vaults, err := local.ListVaults(context.Background())
	require.NoError(t, err)
	assert.Empty(t, vaults, "cache from the previous account must be wiped on register")
}

// TestReconcileAccount_EmptyUserID_NoOp: сервер без UserID в ответе — не должен вызывать сброс кеша.
func TestReconcileAccount_EmptyUserID_NoOp(t *testing.T) {
	local := memStore()
	seedVault(t, local, "vault-1")

	res := contracts.LoginResult{Tokens: contracts.Tokens{AccessToken: "a", RefreshToken: "r"}} // UserID пуст
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(res, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(res.Tokens).Return(nil)

	uc := authuc.New(server, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, store, session.New(), local)
	require.NoError(t, uc.Login(context.Background(), "alice", []byte("pw")))

	vaults, err := local.ListVaults(context.Background())
	require.NoError(t, err)
	assert.Len(t, vaults, 1, "empty user_id must not trigger a wipe")
}

func seedVault(t *testing.T, local *localstore.Store, id string) {
	t.Helper()
	require.NoError(t, local.UpsertVault(context.Background(), contracts.LocalVault{
		ID: id, WrappedVaultKey: []byte("wrapped"), EncName: []byte("name"), Version: 1,
	}))
}
