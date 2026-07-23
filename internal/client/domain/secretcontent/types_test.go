package secretcontent_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func newVaultKey(t *testing.T) []byte {
	t.Helper()
	vaultKey := make([]byte, crypto.KeySize)
	_, err := rand.Read(vaultKey)
	require.NoError(t, err)
	return vaultKey
}

func TestTextTiers_EncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()
	c := cryptoimpl.Crypto{}
	vaultKey := newVaultKey(t)

	row := secretcontent.TextRow{V: secretcontent.TextSchemaV1, Title: "Meeting notes", Tags: []string{"work"}}
	index := secretcontent.TextIndex{V: secretcontent.TextSchemaV1, Note: "quarterly review", CustomFields: []secretcontent.KeyValue{{Key: "project", Value: "alpha"}}}
	payload := secretcontent.TextPayload{V: secretcontent.TextSchemaV1, Body: "long text body...", OTPCodes: []secretcontent.OTPCode{{Code: "111222", Used: false}}}

	encRow, err := c.EncryptStruct(vaultKey, nil, row)
	require.NoError(t, err)
	encIndex, err := c.EncryptStruct(vaultKey, nil, index)
	require.NoError(t, err)
	encPayload, err := c.EncryptStruct(vaultKey, nil, payload)
	require.NoError(t, err)

	var gotRow secretcontent.TextRow
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encRow, &gotRow))
	assert.Equal(t, row, gotRow)

	var gotIndex secretcontent.TextIndex
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encIndex, &gotIndex))
	assert.Equal(t, index, gotIndex)

	var gotPayload secretcontent.TextPayload
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encPayload, &gotPayload))
	assert.Equal(t, payload, gotPayload)
}

func TestBankCardTiers_EncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()
	c := cryptoimpl.Crypto{}
	vaultKey := newVaultKey(t)

	row := secretcontent.BankCardRow{V: secretcontent.BankCardSchemaV1, Title: "Work card", Last4: "4412"}
	index := secretcontent.BankCardIndex{
		V: secretcontent.BankCardSchemaV1, Bank: "Alfa", Cardholder: "IVAN IVANOV",
		Brand: "Visa", Expiry: "08/27", Note: "corporate",
	}
	payload := secretcontent.BankCardPayload{V: secretcontent.BankCardSchemaV1, PAN: "4276000000004412", CVV: "123", PIN: "4321"}

	encRow, err := c.EncryptStruct(vaultKey, nil, row)
	require.NoError(t, err)
	encIndex, err := c.EncryptStruct(vaultKey, nil, index)
	require.NoError(t, err)
	encPayload, err := c.EncryptStruct(vaultKey, nil, payload)
	require.NoError(t, err)

	var gotRow secretcontent.BankCardRow
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encRow, &gotRow))
	assert.Equal(t, row, gotRow)

	var gotIndex secretcontent.BankCardIndex
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encIndex, &gotIndex))
	assert.Equal(t, index, gotIndex)

	var gotPayload secretcontent.BankCardPayload
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encPayload, &gotPayload))
	assert.Equal(t, payload, gotPayload)
}

func TestTOTPTiers_EncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()
	c := cryptoimpl.Crypto{}
	vaultKey := newVaultKey(t)

	row := secretcontent.TOTPRow{V: secretcontent.TOTPSchemaV1, Title: "GitHub 2FA", Issuer: "GitHub"}
	index := secretcontent.TOTPIndex{V: secretcontent.TOTPSchemaV1, Account: "alice@example.com", Note: "backup device configured"}
	payload := secretcontent.TOTPPayload{V: secretcontent.TOTPSchemaV1, Secret: "JBSWY3DPEHPK3PXP", Algo: "SHA1", Digits: 6, Period: 30}

	encRow, err := c.EncryptStruct(vaultKey, nil, row)
	require.NoError(t, err)
	encIndex, err := c.EncryptStruct(vaultKey, nil, index)
	require.NoError(t, err)
	encPayload, err := c.EncryptStruct(vaultKey, nil, payload)
	require.NoError(t, err)

	var gotRow secretcontent.TOTPRow
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encRow, &gotRow))
	assert.Equal(t, row, gotRow)

	var gotIndex secretcontent.TOTPIndex
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encIndex, &gotIndex))
	assert.Equal(t, index, gotIndex)

	var gotPayload secretcontent.TOTPPayload
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encPayload, &gotPayload))
	assert.Equal(t, payload, gotPayload)
}

func TestBinaryTiers_EncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()
	c := cryptoimpl.Crypto{}
	vaultKey := newVaultKey(t)

	row := secretcontent.BinaryRow{V: secretcontent.BinarySchemaV1, Title: "photo.png", Filename: "photo.png"}
	index := secretcontent.BinaryIndex{V: secretcontent.BinarySchemaV1, Size: 123456, Mime: "image/png", Note: "vacation photo"}
	payload := secretcontent.BinaryPayload{V: secretcontent.BinarySchemaV1}

	encRow, err := c.EncryptStruct(vaultKey, nil, row)
	require.NoError(t, err)
	encIndex, err := c.EncryptStruct(vaultKey, nil, index)
	require.NoError(t, err)
	encPayload, err := c.EncryptStruct(vaultKey, nil, payload)
	require.NoError(t, err)

	var gotRow secretcontent.BinaryRow
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encRow, &gotRow))
	assert.Equal(t, row, gotRow)

	var gotIndex secretcontent.BinaryIndex
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encIndex, &gotIndex))
	assert.Equal(t, index, gotIndex)

	var gotPayload secretcontent.BinaryPayload
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encPayload, &gotPayload))
	assert.Equal(t, payload, gotPayload)
}

// TestLoginPasswordPayload_OTPCodes проверяет, что otp_codes доступны у ЛЮБОГО типа секрета,
// не только totp — здесь для login_password.
func TestLoginPasswordPayload_OTPCodes(t *testing.T) {
	t.Parallel()
	c := cryptoimpl.Crypto{}
	vaultKey := newVaultKey(t)

	payload := secretcontent.LoginPasswordPayload{
		V: secretcontent.LoginPasswordSchemaV1, Password: "hunter2",
		OTPCodes: []secretcontent.OTPCode{{Code: "AAAA-1111", Used: false}, {Code: "BBBB-2222", Used: true}},
	}
	enc, err := c.EncryptStruct(vaultKey, nil, payload)
	require.NoError(t, err)

	var got secretcontent.LoginPasswordPayload
	require.NoError(t, c.DecryptStruct(vaultKey, nil, enc, &got))
	assert.Equal(t, payload, got)
	assert.Len(t, got.OTPCodes, 2)
	assert.True(t, got.OTPCodes[1].Used)
}
