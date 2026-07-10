package secret_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func testCipher() cryptoimpl.Crypto { return cryptoimpl.Crypto{} }

func TestCreateText_CreatesAndLists(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(2), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	uc := newSecretUC(t, server, sess)
	id, err := uc.CreateText(context.Background(), "v1", secret.CreateTextInput{Title: "Note", Body: "hello"})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	rows, err := uc.ListTextRows(context.Background(), "v1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Note", rows[0].Row.Title)
}

func TestCreateText_EmptyTitle(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	uc := newSecretUC(t, mocks.NewMockServerClient(t), sess)
	_, err := uc.CreateText(context.Background(), "v1", secret.CreateTextInput{})
	require.ErrorIs(t, err, secret.ErrEmptyTitle)
}

func TestCreateBankCard_Last4DerivedFromPAN(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(4), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	uc := newSecretUC(t, server, sess)
	_, err := uc.CreateBankCard(context.Background(), "v1", secret.CreateBankCardInput{
		Title: "Work card", PAN: "4276000000004412", CVV: "123",
	})
	require.NoError(t, err)

	rows, err := uc.ListBankCardRows(context.Background(), "v1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "4412", rows[0].Row.Last4)
}

func TestCreateTOTP_RequiresSecret(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	uc := newSecretUC(t, mocks.NewMockServerClient(t), sess)
	_, err := uc.CreateTOTP(context.Background(), "v1", secret.CreateTOTPInput{Title: "GitHub"})
	require.ErrorIs(t, err, secret.ErrEmptyTOTPSecret)
}

func TestCreateTOTP_CreatesAndGeneratesCode(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(5), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	uc := newSecretUC(t, server, sess)
	id, err := uc.CreateTOTP(context.Background(), "v1", secret.CreateTOTPInput{
		Title: "GitHub", Issuer: "GitHub", Secret: "JBSWY3DPEHPK3PXP",
	})
	require.NoError(t, err)

	// GetTOTPDetail требует GetSecretPayload на сервере (авторитетный источник существования).
	rows, err := uc.ListTOTPRows(context.Background(), "v1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, id, rows[0].ID)

	code, err := secret.GenerateTOTPCode(secretcontentPayload(t))
	require.NoError(t, err)
	assert.Len(t, code, 6)
}

func secretcontentPayload(t *testing.T) secretcontent.TOTPPayload {
	t.Helper()
	return secretcontent.TOTPPayload{Secret: "JBSWY3DPEHPK3PXP", Digits: 6, Period: 30}
}

func TestParseOTPAuthURI_Valid(t *testing.T) {
	in, err := secret.ParseOTPAuthURI("otpauth://totp/GitHub:alice@example.com?secret=JBSWY3DPEHPK3PXP&issuer=GitHub&digits=6&period=30")
	require.NoError(t, err)
	assert.Equal(t, "GitHub", in.Issuer)
	assert.Equal(t, "alice@example.com", in.Account)
	assert.Equal(t, "JBSWY3DPEHPK3PXP", in.Secret)
	assert.Equal(t, 6, in.Digits)
	assert.Equal(t, 30, in.Period)
}

func TestParseOTPAuthURI_Invalid(t *testing.T) {
	_, err := secret.ParseOTPAuthURI("not a uri")
	require.ErrorIs(t, err, secret.ErrInvalidOTPAuthURI)
}

// TestUpdateText_Conflict проверяет GenericConflict для не-login_password типа.
func TestUpdateText_Conflict(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	server := mocks.NewMockServerClient(t)

	serverEncRow, err := encryptTestStruct(vaultKey, "v1", "s1", 5, secret.TierRow, secretcontent.TextRow{V: 1, Title: "ServerNote"})
	require.NoError(t, err)
	serverEncIndex, err := encryptTestStruct(vaultKey, "v1", "s1", 5, secret.TierIndex, secretcontent.TextIndex{V: 1, Note: "server side"})
	require.NoError(t, err)
	serverEncPayload, err := encryptTestStruct(vaultKey, "v1", "s1", 5, secret.TierPayload, secretcontent.TextPayload{V: 1, Body: "server body"})
	require.NoError(t, err)

	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(3), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(0), &grpcclient.ConflictError{Server: contracts.ServerSecret{
			ID: "s1", Type: 2, Version: 5, EncRow: serverEncRow, EncIndex: serverEncIndex, EncPayload: serverEncPayload,
		}})

	uc := newSecretUC(t, server, sess)
	conflict, err := uc.UpdateText(context.Background(), "v1", "s1", 3, secret.CreateTextInput{Title: "MyNote", Body: "my body"})
	require.NoError(t, err)
	require.NotNil(t, conflict)

	assert.Equal(t, "MyNote", conflict.MineRow["title"])
	assert.Equal(t, "ServerNote", conflict.ServerRow["title"])
	assert.Equal(t, "server body", conflict.ServerPayload["body"])
	assert.Equal(t, int64(5), conflict.ServerVersion)
}

func TestUpdateText_EmptyTitle(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	uc := newSecretUC(t, mocks.NewMockServerClient(t), sess)
	_, err := uc.UpdateText(context.Background(), "v1", "s1", 1, secret.CreateTextInput{})
	require.ErrorIs(t, err, secret.ErrEmptyTitle)
}

// TestCreateText_IndexTooLarge проверяет ErrIndexTooLarge при превышении лимита enc_index.
func TestCreateText_IndexTooLarge(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	uc := newSecretUC(t, mocks.NewMockServerClient(t), sess) // CreateSecret не должен вызываться

	hugeNote := strings.Repeat("a", 9*1024) // 9 KiB > 8 KiB лимита
	_, err := uc.CreateText(context.Background(), "v1", secret.CreateTextInput{Title: "Note", Note: hugeNote, Body: "x"})
	require.ErrorIs(t, err, secret.ErrIndexTooLarge)
}

// encryptTestStruct — хелпер для шифрования серверной версии произвольного тира в тестах конфликта.
func encryptTestStruct(vaultKey []byte, vaultID, secretID string, version int64, tier string, value any) ([]byte, error) {
	c := testCipher()
	return c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, secretID, version, tier), value)
}
