package secret_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func TestListAllRows_EmptyVaultID(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	sess, _ := openVaultSession(t, "v1")
	_, err := newSecretUC(t, server, sess).ListAllRows(context.Background(), "")
	require.ErrorIs(t, err, secret.ErrEmptyVaultID)
}

func TestListAllRows_MixedTypes(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}

	loginRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierRow),
		secretcontent.LoginPasswordRow{V: 1, Title: "GitHub", Username: "alice", URI: "github.com"})
	require.NoError(t, err)
	totpRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s2", 1, secret.TierRow),
		secretcontent.TOTPRow{V: 1, Title: "AWS", Issuer: "Amazon"})
	require.NoError(t, err)

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: int32(domain.SecretTypeLoginPassword), EncRow: loginRow, Version: 1,
	}))
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s2", VaultID: "v1", Type: int32(domain.SecretTypeTOTP), EncRow: totpRow, Version: 1,
	}))

	server := mocks.NewMockServerClient(t)
	rows, err := newSecretUCStore(t, server, sess, local).ListAllRows(context.Background(), "v1")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byID := map[string]secret.SummaryRow{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	assert.Equal(t, "GitHub", byID["s1"].Title)
	assert.Equal(t, "alice", byID["s1"].Subtitle)
	assert.Equal(t, "github.com", byID["s1"].URI)
	assert.Equal(t, "AWS", byID["s2"].Title)
	assert.Equal(t, "Amazon", byID["s2"].Subtitle)
}

func TestListRowsByType_Filters(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}

	loginRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierRow),
		secretcontent.LoginPasswordRow{V: 1, Title: "GitHub"})
	require.NoError(t, err)
	textRow, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s2", 1, secret.TierRow),
		secretcontent.TextRow{V: 1, Title: "Note"})
	require.NoError(t, err)

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: int32(domain.SecretTypeLoginPassword), EncRow: loginRow, Version: 1,
	}))
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s2", VaultID: "v1", Type: int32(domain.SecretTypeText), EncRow: textRow, Version: 1,
	}))

	server := mocks.NewMockServerClient(t)
	rows, err := newSecretUCStore(t, server, sess, local).ListRowsByType(context.Background(), "v1", domain.SecretTypeText)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Note", rows[0].Title)
}

func TestListAllRows_BankCardWithIndexLoaded(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}

	row, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierRow),
		secretcontent.BankCardRow{V: 1, Title: "My Card", Last4: "4242"})
	require.NoError(t, err)
	idx, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierIndex),
		secretcontent.BankCardIndex{V: 1, Expiry: "12/29", Bank: "Chase", Cardholder: "Alice", Brand: "Visa"})
	require.NoError(t, err)

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: int32(domain.SecretTypeBankCard), EncRow: row, Version: 1,
	}))
	require.NoError(t, local.SetSecretIndex(context.Background(), "s1", idx, 1))

	server := mocks.NewMockServerClient(t)
	rows, err := newSecretUCStore(t, server, sess, local).ListAllRows(context.Background(), "v1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "12/29", rows[0].Expiry)
	assert.Equal(t, "Chase", rows[0].Bank)
	assert.Equal(t, "Alice", rows[0].Cardholder)
	assert.Equal(t, "Visa", rows[0].Brand)
	assert.Equal(t, "•• 4242", rows[0].Subtitle)
}

func TestListAllRows_BinaryWithIndexLoaded(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}

	row, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierRow),
		secretcontent.BinaryRow{V: 1, Title: "Doc", Filename: "doc.pdf"})
	require.NoError(t, err)
	idx, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierIndex),
		secretcontent.BinaryIndex{V: 1, Size: 2048})
	require.NoError(t, err)

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: int32(domain.SecretTypeBinary), EncRow: row, Version: 1,
	}))
	require.NoError(t, local.SetSecretIndex(context.Background(), "s1", idx, 1))

	server := mocks.NewMockServerClient(t)
	rows, err := newSecretUCStore(t, server, sess, local).ListAllRows(context.Background(), "v1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "doc.pdf", rows[0].Subtitle)
	assert.Equal(t, int64(2048), rows[0].Size)
}

func TestListAllRows_SkipsUndecryptableRow(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: int32(domain.SecretTypeLoginPassword), EncRow: []byte("garbage"), Version: 1,
	}))

	server := mocks.NewMockServerClient(t)
	rows, err := newSecretUCStore(t, server, sess, local).ListAllRows(context.Background(), "v1")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestListAllRows_VaultLocked(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	local := newMemStore(t)
	_, err := newSecretUCStore(t, server, session.New(), local).ListAllRows(context.Background(), "v1")
	require.Error(t, err)
}
