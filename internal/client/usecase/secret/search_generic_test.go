package secret_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// TestSearch_WorksAcrossSecretTypes проверяет, что поиск (типонезависимый)
// находит секреты не только type=login_password, но и bank_card/text — по Tier 2a (title/last4)
// и по Tier 2b (bank/note) после догрузки индекса.
func TestSearch_WorksAcrossSecretTypes(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(4), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(2), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	uc := newSecretUC(t, server, sess)

	_, err := uc.CreateBankCard(context.Background(), "v1", secret.CreateBankCardInput{
		Title: "Work Visa", PAN: "4276000000004412", Bank: "Alfa-Bank", Note: "corporate travel card",
	})
	require.NoError(t, err)

	_, err = uc.CreateText(context.Background(), "v1", secret.CreateTextInput{
		Title: "Wifi password backup", Body: "irrelevant", Note: "office router credentials",
	})
	require.NoError(t, err)

	// Поиск по Tier 2a bank_card (last4 из PAN).
	res, err := uc.Search(context.Background(), "v1", "4412")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "Work Visa", res.Rows[0].Row.Title)

	// Поиск по Tier 2a title текстовой заметки.
	res, err = uc.Search(context.Background(), "v1", "wifi")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "Wifi password backup", res.Rows[0].Row.Title)

	// Онлайн-создание кеширует Tier 2b сразу (index_loaded=1) — Tier 2b-поле
	// bank_card.bank (в enc_index) находится сразу, без отдельной LoadIndexes.
	res, err = uc.Search(context.Background(), "v1", "alfa-bank")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.False(t, res.Incomplete)
}

// TestSearch_MatchesCustomFieldsAcrossTypes проверяет поиск по custom_fields
func TestSearch_MatchesCustomFieldsAcrossTypes(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(4), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	uc := newSecretUC(t, server, sess)

	_, err := uc.CreateBankCard(context.Background(), "v1", secret.CreateBankCardInput{
		Title: "Work Visa", PAN: "4276000000004412", Bank: "Alfa-Bank", Note: "corporate travel card",
	})
	require.NoError(t, err)

	// index_loaded=1 сразу после онлайн-создания (encIndex непустой) — поиск по note/bank работает
	// без отдельного LoadIndexes и результат помечается полным (Incomplete=false).
	res, err := uc.Search(context.Background(), "v1", "alfa-bank")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.False(t, res.Incomplete)

	res, err = uc.Search(context.Background(), "v1", "corporate travel")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
}
