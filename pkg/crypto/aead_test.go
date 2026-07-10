package crypto_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func mustKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, crypto.KeySize)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	plaintext := []byte("super secret vault key material")

	blob, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)

	got, err := crypto.Decrypt(key, blob)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestEncrypt_EmptyPlaintextRoundTrips(t *testing.T) {
	t.Parallel()

	key := mustKey(t)

	blob, err := crypto.Encrypt(key, nil)
	require.NoError(t, err)

	got, err := crypto.Decrypt(key, blob)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestEncrypt_NonDeterministic: один plaintext на один ключ даёт разные блобы
// (разный случайный nonce), т.е. шифрование не детерминировано.
func TestEncrypt_NonDeterministic(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	plaintext := []byte("same input")

	blob1, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)
	blob2, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, blob1, blob2, "ciphertext must differ across calls")
	assert.NotEqual(t, blob1[:crypto.NonceSize], blob2[:crypto.NonceSize], "nonce must differ across calls")
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	t.Parallel()

	blob, err := crypto.Encrypt(mustKey(t), []byte("secret"))
	require.NoError(t, err)

	_, err = crypto.Decrypt(mustKey(t), blob)
	require.Error(t, err, "decrypt with a different key must fail, not panic")
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	blob, err := crypto.Encrypt(key, []byte("secret"))
	require.NoError(t, err)

	// Портим последний байт (тег Poly1305).
	blob[len(blob)-1] ^= 0xff

	_, err = crypto.Decrypt(key, blob)
	require.Error(t, err)
}

func TestDecrypt_TooShortBlob(t *testing.T) {
	t.Parallel()

	_, err := crypto.Decrypt(mustKey(t), []byte("short"))
	require.ErrorIs(t, err, crypto.ErrCiphertextTooShort)
}

func TestEncryptDecrypt_InvalidKeySize(t *testing.T) {
	t.Parallel()

	_, err := crypto.Encrypt(make([]byte, crypto.KeySize-1), []byte("x"))
	require.ErrorIs(t, err, crypto.ErrInvalidKeySize)

	_, err = crypto.Decrypt(make([]byte, crypto.KeySize+1), make([]byte, crypto.NonceSize+16))
	require.ErrorIs(t, err, crypto.ErrInvalidKeySize)
}

// TestWrapUnwrap_RoundTrip: envelope-обёртка — VaultKey под MasterKey.
func TestWrapUnwrap_RoundTrip(t *testing.T) {
	t.Parallel()

	kek := mustKey(t)
	vaultKey := mustKey(t)

	wrapped, err := crypto.WrapKey(kek, vaultKey)
	require.NoError(t, err)
	assert.False(t, bytes.Equal(wrapped, vaultKey), "wrapped key must not equal plaintext key")

	unwrapped, err := crypto.UnwrapKey(kek, wrapped)
	require.NoError(t, err)
	assert.Equal(t, vaultKey, unwrapped)
}

func TestUnwrapKey_WrongKEKFails(t *testing.T) {
	t.Parallel()

	wrapped, err := crypto.WrapKey(mustKey(t), mustKey(t))
	require.NoError(t, err)

	_, err = crypto.UnwrapKey(mustKey(t), wrapped)
	require.Error(t, err)
}

// TestEncryptDecryptWithAD_RoundTrip: расшифровка с той же AD проходит.
func TestEncryptDecryptWithAD_RoundTrip(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	plaintext := []byte("secret bound to context")
	ad := []byte("vault-1|secret-1|v3")

	blob, err := crypto.EncryptWithAD(key, plaintext, ad)
	require.NoError(t, err)

	got, err := crypto.DecryptWithAD(key, blob, ad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

// TestDecryptWithAD_WrongADFails: подмена контекста (другой secret_id/version) ломает расшифровку.
func TestDecryptWithAD_WrongADFails(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	blob, err := crypto.EncryptWithAD(key, []byte("secret"), []byte("vault-1|secret-1|v3"))
	require.NoError(t, err)

	// Другой secret_id.
	_, err = crypto.DecryptWithAD(key, blob, []byte("vault-1|secret-2|v3"))
	require.Error(t, err, "decrypt with a different secret_id in AD must fail")

	// Откат версии.
	_, err = crypto.DecryptWithAD(key, blob, []byte("vault-1|secret-1|v2"))
	require.Error(t, err, "decrypt with a rolled-back version in AD must fail")
}

// TestDecryptWithAD_MissingADFails: блоб зашифрован с AD, расшифровка без AD не проходит.
func TestDecryptWithAD_MissingADFails(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	ad := []byte("vault-1|secret-1|v3")
	blob, err := crypto.EncryptWithAD(key, []byte("secret"), ad)
	require.NoError(t, err)

	_, err = crypto.Decrypt(key, blob)
	require.Error(t, err, "AD-bound ciphertext must not decrypt without AD")
}

// TestEncrypt_NilADEqualsEmptyAD: Encrypt == EncryptWithAD(..., nil) по совместимости.
func TestEncrypt_NilADEqualsEmptyAD(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	blob, err := crypto.Encrypt(key, []byte("secret"))
	require.NoError(t, err)

	got, err := crypto.DecryptWithAD(key, blob, nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("secret"), got)
}
