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
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func newSecretUC(t *testing.T, server contracts.ServerClient, sess *session.Session) *secret.UseCase {
	return newSecretUCStore(t, server, sess, newMemStore(t))
}

func newSecretUCStore(t *testing.T, server contracts.ServerClient, sess *session.Session, local *localstore.Store) *secret.UseCase {
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{AccessToken: "tok"}, nil).Maybe()
	return secret.New(server, cryptoimpl.Crypto{}, store, sess, local)
}

// newMemStore открывает in-memory localstore и закрывает его по завершении теста.
func newMemStore(t *testing.T) *localstore.Store {
	t.Helper()
	ls, err := localstore.Open("", false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ls.Close() })
	return ls
}

// openVaultSession возвращает сессию с открытой папкой vaultID и его VaultKey.
func openVaultSession(t *testing.T, vaultID string) (*session.Session, []byte) {
	t.Helper()
	sess := session.New()
	vk, err := crypto.GenerateKey()
	require.NoError(t, err)
	sess.OpenVault(vaultID, vk)
	return sess, vk
}

func TestCreateLoginPassword_EncryptsTiers(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "vault-1")

	var gotID string
	var gotRow, gotPayload []byte
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "vault-1", int32(1), mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _, secretID, _ string, _ int32, encRow, _, encPayload []byte) error {
			gotID, gotRow, gotPayload = secretID, encRow, encPayload
			return nil
		})

	id, err := newSecretUC(t, server, sess).CreateLoginPassword(context.Background(), "vault-1", secret.CreateLoginPasswordInput{
		Title:    "GitHub",
		Username: "alice",
		Password: "hunter2",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, id, gotID, "клиент генерирует secret_id и шлёт его на сервер")

	// enc_row и enc_payload расшифровываются VaultKey'ом с AAD-контекстом (vault|secret|version|tier).
	c := cryptoimpl.Crypto{}
	var row secretcontent.LoginPasswordRow
	require.NoError(t, c.DecryptStruct(vaultKey, secret.SecretAAD("vault-1", id, 1, secret.TierRow), gotRow, &row))
	assert.Equal(t, "GitHub", row.Title)
	assert.Equal(t, "alice", row.Username)

	var payload secretcontent.LoginPasswordPayload
	require.NoError(t, c.DecryptStruct(vaultKey, secret.SecretAAD("vault-1", id, 1, secret.TierPayload), gotPayload, &payload))
	assert.Equal(t, "hunter2", payload.Password)
}

func TestCreateLoginPassword_VaultLocked(t *testing.T) {
	server := mocks.NewMockServerClient(t) // RPC не должен вызываться
	_, err := newSecretUC(t, server, session.New()).CreateLoginPassword(context.Background(), "vault-1", secret.CreateLoginPasswordInput{Title: "x"})
	require.ErrorIs(t, err, secret.ErrVaultLocked)
}

func TestCreateLoginPassword_Validation(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	sess, _ := openVaultSession(t, "vault-1")
	uc := newSecretUC(t, server, sess)

	_, err := uc.CreateLoginPassword(context.Background(), "", secret.CreateLoginPasswordInput{Title: "x"})
	require.ErrorIs(t, err, secret.ErrEmptyVaultID)

	_, err = uc.CreateLoginPassword(context.Background(), "vault-1", secret.CreateLoginPasswordInput{Title: ""})
	require.ErrorIs(t, err, secret.ErrEmptyTitle)
}

func TestListRow_ReadsFromLocalStore(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "vault-1")

	c := cryptoimpl.Crypto{}
	encRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD("vault-1", "s1", 1, secret.TierRow), secretcontent.LoginPasswordRow{
		V: secretcontent.LoginPasswordSchemaV1, Title: "GitHub", Username: "alice",
	})
	require.NoError(t, err)

	// Наполняем локальный кеш напрямую — ListRow не должен ходить в сеть.
	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "vault-1", Type: 1, EncRow: encRow, Version: 1,
	}))

	server := mocks.NewMockServerClient(t) // ListSecretRows не должен вызываться
	got, err := newSecretUCStore(t, server, sess, local).ListRow(context.Background(), "vault-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "GitHub", got[0].Row.Title)
	assert.Equal(t, "alice", got[0].Row.Username)
}

func TestGetPayload_Decrypts(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "vault-1")

	c := cryptoimpl.Crypto{}
	encPayload, err := c.EncryptStruct(vaultKey, secret.SecretAAD("vault-1", "s1", 1, secret.TierPayload), secretcontent.LoginPasswordPayload{
		V: secretcontent.LoginPasswordSchemaV1, Password: "hunter2",
	})
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().GetSecretPayload(mock.Anything, "tok", "s1").Return(contracts.SecretPayloadItem{
		ID: "s1", Type: 1, Version: 1, EncPayload: encPayload,
	}, nil)

	got, err := newSecretUC(t, server, sess).GetPayload(context.Background(), "vault-1", "s1")
	require.NoError(t, err)
	assert.Equal(t, "hunter2", got.Payload.Password)
}

func TestGetPayload_VaultLocked(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	_, err := newSecretUC(t, server, session.New()).GetPayload(context.Background(), "vault-1", "s1")
	require.ErrorIs(t, err, secret.ErrVaultLocked)
}
