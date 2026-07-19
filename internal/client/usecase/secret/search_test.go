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
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// seedRow кладёт секрет (Tier 2a, index не загружен) в локальный кеш.
func seedRow(t *testing.T, local *localstore.Store, vaultKey []byte, vaultID, id, title, username string) {
	t.Helper()
	c := cryptoimpl.Crypto{}
	encRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, id, 1, secret.TierRow),
		secretcontent.LoginPasswordRow{V: 1, Title: title, Username: username})
	require.NoError(t, err)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: id, VaultID: vaultID, Type: 1, EncRow: encRow, Version: 1,
	}))
}

// TestSearch_Tier2aAlways_IncompleteFlag: до догрузки Tier 2b поиск идёт по Tier 2a и помечается неполным.
func TestSearch_Tier2aAlways_IncompleteFlag(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	local := newMemStore(t)
	seedRow(t, local, vaultKey, "v1", "s1", "GitHub", "alice")
	seedRow(t, local, vaultKey, "v1", "s2", "Gmail", "bob")

	uc := newSecretUCStore(t, mocks.NewMockServerClient(t), sess, local)

	// Поиск по Tier 2a (title) — находит, но флаг Incomplete выставлен (Tier 2b не загружен).
	res, err := uc.Search(context.Background(), "v1", "github")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "GitHub", res.Rows[0].Row.Title)
	assert.True(t, res.Incomplete, "Tier 2b ещё не догружен — поиск неполный")

	// Поиск по note (Tier 2b) пока не срабатывает — индекс не загружен.
	res, err = uc.Search(context.Background(), "v1", "server-note")
	require.NoError(t, err)
	assert.Empty(t, res.Rows)
	assert.True(t, res.Incomplete)
}

// TestLoadIndexes_ThenSearchByNote: после фоновой догрузки Tier 2b поиск по note срабатывает,
// а флаг Incomplete снимается.
func TestLoadIndexes_ThenSearchByNote(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	local := newMemStore(t)
	seedRow(t, local, vaultKey, "v1", "s1", "GitHub", "alice")

	// Серверный индекс для s1 с note "backup-codes".
	c := cryptoimpl.Crypto{}
	encIndex, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierIndex),
		secretcontent.LoginPasswordIndex{V: 1, Note: "backup-codes"})
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListSecretIndex(mock.Anything, "tok", "v1").
		Return([]contracts.SecretIndexItem{{ID: "s1", Version: 1, EncIndex: encIndex}}, nil)

	uc := newSecretUCStore(t, server, sess, local)

	// До догрузки — поиск по note пуст и неполон.
	res, err := uc.Search(context.Background(), "v1", "backup")
	require.NoError(t, err)
	assert.Empty(t, res.Rows)
	assert.True(t, res.Incomplete)

	// Фоновая догрузка Tier 2b.
	require.NoError(t, uc.LoadIndexes(context.Background(), "v1"))

	// Теперь поиск по note срабатывает и результат полон.
	res, err = uc.Search(context.Background(), "v1", "backup")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "s1", res.Rows[0].ID)
	assert.False(t, res.Incomplete)
}

// Регрессия: один секрет с повреждённым/рассинхронизированным шифротекстом (например после
// гонки конфликтов синхронизации — enc_row остался под старой версией/AAD) не должен обрывать
// поиск по всей папке.
func TestSearch_SkipsBrokenSecret_KeepsOthers(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	local := newMemStore(t)
	seedRow(t, local, vaultKey, "v1", "s1", "GitHub", "alice")
	seedRow(t, local, vaultKey, "v1", "s2", "Gmail", "bob")

	// s3 — битый секрет: enc_row зашифрован под ДРУГУЮ версию, чем та, что хранится в кеше
	// (эмулирует рассинхрон AAD/version после гонки конфликтов).
	c := cryptoimpl.Crypto{}
	staleEncRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s3", 1, secret.TierRow),
		secretcontent.LoginPasswordRow{V: 1, Title: "Broken"})
	require.NoError(t, err)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s3", VaultID: "v1", Type: 1, EncRow: staleEncRow, Version: 2, // версия не совпадает с AAD (1)
	}))

	uc := newSecretUCStore(t, mocks.NewMockServerClient(t), sess, local)

	res, err := uc.Search(context.Background(), "v1", "")
	require.NoError(t, err, "битый секрет не должен приводить к ошибке всего поиска")
	require.Len(t, res.Rows, 2, "здоровые секреты (s1, s2) должны остаться в результате")

}
