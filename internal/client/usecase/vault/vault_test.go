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
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// Реальный cipher (cryptoimpl) + mock ServerClient + mock TokenStore + реальная сессия + in-memory localstore.
func newVaultUC(t *testing.T, server contracts.ServerClient, sess *session.Session) *vault.UseCase {
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{AccessToken: "tok"}, nil).Maybe()
	return vault.New(server, cryptoimpl.Crypto{}, store, sess, newMemStore(t))
}

// newMemStore открывает in-memory localstore и закрывает его по завершении теста.
func newMemStore(t *testing.T) *localstore.Store {
	t.Helper()
	ls, err := localstore.Open("", false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ls.Close() })
	return ls
}

func unlockedSession(t *testing.T) *session.Session {
	t.Helper()
	sess := session.New()
	mk, err := crypto.GenerateKey()
	require.NoError(t, err)
	sess.SetMasterKey(mk)
	return sess
}

func TestCreate_Success_EncryptsAndOpensSession(t *testing.T) {
	sess := unlockedSession(t)
	mk, _ := sess.MasterKey()

	var gotWrapped, gotEncName []byte
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateVault(mock.Anything, "tok", mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, wrapped, encName []byte) (string, error) {
			gotWrapped, gotEncName = wrapped, encName
			return "vault-1", nil
		})

	id, err := newVaultUC(t, server, sess).Create(context.Background(), "Personal")
	require.NoError(t, err)
	assert.Equal(t, "vault-1", id)

	// Папка открыт в сессии.
	vk, ok := sess.VaultKey("vault-1")
	require.True(t, ok)

	// E2E-проверка: сервер получил обёртки, из которых разворачивается корректный ключ и имя.
	c := cryptoimpl.Crypto{}
	unwrapped, err := c.UnwrapVaultKey(gotWrapped, mk)
	require.NoError(t, err)
	assert.Equal(t, vk, unwrapped)

	var name string
	require.NoError(t, c.DecryptStruct(unwrapped, gotEncName, &name))
	assert.Equal(t, "Personal", name)
}

func TestCreate_Locked(t *testing.T) {
	server := mocks.NewMockServerClient(t) // RPC не должен вызываться
	_, err := newVaultUC(t, server, session.New()).Create(context.Background(), "Personal")
	require.ErrorIs(t, err, vault.ErrLocked)
}

func TestCreate_EmptyName(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	_, err := newVaultUC(t, server, unlockedSession(t)).Create(context.Background(), "")
	require.ErrorIs(t, err, vault.ErrEmptyName)
}

func TestList_DecryptsAndOpensSession(t *testing.T) {
	sess := unlockedSession(t)
	mk, _ := sess.MasterKey()

	// Готовим серверный ответ: папка, зашифрованная тем же MasterKey.
	c := cryptoimpl.Crypto{}
	vaultKey, err := c.GenerateVaultKey()
	require.NoError(t, err)
	wrapped, err := c.WrapVaultKey(vaultKey, mk)
	require.NoError(t, err)
	encName, err := c.EncryptStruct(vaultKey, "Work")
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListVaults(mock.Anything, "tok").Return([]contracts.VaultItem{
		{ID: "v1", WrappedVaultKey: wrapped, EncName: encName, Version: 2},
	}, nil)

	got, err := newVaultUC(t, server, sess).List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Work", got[0].Name)
	assert.Equal(t, int64(2), got[0].Version)

	_, ok := sess.VaultKey("v1")
	assert.True(t, ok, "vault must be opened in session")
}

func TestList_Locked(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	_, err := newVaultUC(t, server, session.New()).List(context.Background())
	require.ErrorIs(t, err, vault.ErrLocked)
}
