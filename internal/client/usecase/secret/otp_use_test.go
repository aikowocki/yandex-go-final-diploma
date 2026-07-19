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
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// otpSecretBlobs шифрует все три тира TOTP-секрета с otp_codes payload под vaultKey.
func otpSecretBlobs(t *testing.T, vaultKey []byte, vaultID, secretID string, version int64, used []bool) (row, index, payload []byte) {
	t.Helper()
	c := cryptoimpl.Crypto{}
	var err error
	row, err = c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, secretID, version, secret.TierRow),
		secretcontent.TOTPRow{V: 1, Title: "Recovery"})
	require.NoError(t, err)
	index, err = c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, secretID, version, secret.TierIndex),
		secretcontent.TOTPIndex{V: 1})
	require.NoError(t, err)

	codes := make([]secretcontent.OTPCode, len(used))
	for i, u := range used {
		codes[i] = secretcontent.OTPCode{Code: "code", Used: u}
	}
	payload, err = c.EncryptStruct(vaultKey, secret.SecretAAD(vaultID, secretID, version, secret.TierPayload),
		secretcontent.TOTPPayload{V: 1, Secret: "SECRET", OTPCodes: codes})
	require.NoError(t, err)
	return row, index, payload
}

func TestMarkOTPCodeUsed_EmptySecretID(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	sess, _ := openVaultSession(t, "v1")
	err := newSecretUC(t, server, sess).MarkOTPCodeUsed(context.Background(), "v1", "", 0)
	require.ErrorIs(t, err, secret.ErrEmptySecretID)
}

func TestMarkOTPCodeUsed_NegativeIndex(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	sess, _ := openVaultSession(t, "v1")
	err := newSecretUC(t, server, sess).MarkOTPCodeUsed(context.Background(), "v1", "s1", -1)
	require.ErrorIs(t, err, secret.ErrInvalidOTPCodeIndex)
}

func TestMarkOTPCodeUsed_SecretNotFound(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	sess, _ := openVaultSession(t, "v1")
	err := newSecretUC(t, server, sess).MarkOTPCodeUsed(context.Background(), "v1", "s1", 0)
	require.ErrorIs(t, err, secret.ErrSecretNotFound)
}

func TestMarkOTPCodeUsed_Success(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	row, index, payload := otpSecretBlobs(t, vaultKey, "v1", "s1", 1, []bool{false, false})

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 5, EncRow: row, Version: 1,
	}))
	require.NoError(t, local.SetSecretIndex(context.Background(), "s1", index, 1))
	require.NoError(t, local.SetSecretPayload(context.Background(), "s1", payload, 1))

	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		UpdateSecret(mock.Anything, "tok", "s1", int64(1), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(2), nil)

	err := newSecretUCStore(t, server, sess, local).MarkOTPCodeUsed(context.Background(), "v1", "s1", 0)
	require.NoError(t, err)
}

func TestMarkOTPCodeUsed_InvalidIndexTooLarge(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	row, index, payload := otpSecretBlobs(t, vaultKey, "v1", "s1", 1, []bool{false})

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 5, EncRow: row, Version: 1,
	}))
	require.NoError(t, local.SetSecretIndex(context.Background(), "s1", index, 1))
	require.NoError(t, local.SetSecretPayload(context.Background(), "s1", payload, 1))

	server := mocks.NewMockServerClient(t)
	err := newSecretUCStore(t, server, sess, local).MarkOTPCodeUsed(context.Background(), "v1", "s1", 5)
	require.ErrorIs(t, err, secret.ErrInvalidOTPCodeIndex)
}

func TestMarkOTPCodeUsed_NoOTPCodes(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}
	row, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierRow), secretcontent.TOTPRow{V: 1, Title: "Recovery"})
	require.NoError(t, err)
	index, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierIndex), secretcontent.TOTPIndex{V: 1})
	require.NoError(t, err)
	payload, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierPayload), secretcontent.TOTPPayload{V: 1, Secret: "SECRET"})
	require.NoError(t, err)

	local := newMemStore(t)
	require.NoError(t, local.UpsertSecretRow(context.Background(), contracts.LocalSecret{
		ID: "s1", VaultID: "v1", Type: 5, EncRow: row, Version: 1,
	}))
	require.NoError(t, local.SetSecretIndex(context.Background(), "s1", index, 1))
	require.NoError(t, local.SetSecretPayload(context.Background(), "s1", payload, 1))

	server := mocks.NewMockServerClient(t)
	err = newSecretUCStore(t, server, sess, local).MarkOTPCodeUsed(context.Background(), "v1", "s1", 0)
	require.ErrorIs(t, err, secret.ErrNoOTPCodes)
}

func TestGetOTPCodes_VaultLocked(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	_, err := newSecretUC(t, server, session.New()).GetOTPCodes(context.Background(), "v1", "s1")
	require.Error(t, err)
}

func TestGetOTPCodes_ReturnsCodesFromPayload(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	_, _, payload := otpSecretBlobs(t, vaultKey, "v1", "s1", 1, []bool{false, true})

	server := mocks.NewMockServerClient(t)
	server.EXPECT().GetSecretPayload(mock.Anything, "tok", "s1").Return(contracts.SecretPayloadItem{
		ID: "s1", Type: 5, Version: 1, EncPayload: payload,
	}, nil)

	codes, err := newSecretUC(t, server, sess).GetOTPCodes(context.Background(), "v1", "s1")
	require.NoError(t, err)
	require.Len(t, codes, 2)
	assert.False(t, codes[0].Used)
	assert.True(t, codes[1].Used)
}

func TestGetOTPCodes_NoCodesReturnsNil(t *testing.T) {
	sess, vaultKey := openVaultSession(t, "v1")
	c := cryptoimpl.Crypto{}
	payload, err := c.EncryptStruct(vaultKey, secret.SecretAAD("v1", "s1", 1, secret.TierPayload), secretcontent.TOTPPayload{V: 1, Secret: "SECRET"})
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().GetSecretPayload(mock.Anything, "tok", "s1").Return(contracts.SecretPayloadItem{
		ID: "s1", Type: 5, Version: 1, EncPayload: payload,
	}, nil)

	codes, err := newSecretUC(t, server, sess).GetOTPCodes(context.Background(), "v1", "s1")
	require.NoError(t, err)
	assert.Nil(t, codes)
}
