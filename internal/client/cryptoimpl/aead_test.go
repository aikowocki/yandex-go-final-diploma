package cryptoimpl_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func mustKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, crypto.KeySize)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

type loginPayload struct {
	V        int    `json:"v"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func TestWrapUnwrapVaultKey_RoundTrip(t *testing.T) {
	t.Parallel()

	c := cryptoimpl.Crypto{}
	masterKey := mustKey(t)
	vaultKey := mustKey(t)

	wrapped, err := c.WrapVaultKey(vaultKey, masterKey)
	require.NoError(t, err)
	assert.NotEqual(t, vaultKey, wrapped)

	got, err := c.UnwrapVaultKey(wrapped, masterKey)
	require.NoError(t, err)
	assert.Equal(t, vaultKey, got)
}

func TestUnwrapVaultKey_WrongMasterKeyFails(t *testing.T) {
	t.Parallel()

	c := cryptoimpl.Crypto{}
	wrapped, err := c.WrapVaultKey(mustKey(t), mustKey(t))
	require.NoError(t, err)

	_, err = c.UnwrapVaultKey(wrapped, mustKey(t))
	require.Error(t, err)
}

func TestEncryptDecryptStruct_RoundTrip(t *testing.T) {
	t.Parallel()

	c := cryptoimpl.Crypto{}
	key := mustKey(t)
	in := loginPayload{V: 1, Username: "alice", Password: "hunter2"}

	blob, err := c.EncryptStruct(key, in)
	require.NoError(t, err)

	var out loginPayload
	require.NoError(t, c.DecryptStruct(key, blob, &out))
	assert.Equal(t, in, out)
}

func TestDecryptStruct_WrongKeyFails(t *testing.T) {
	t.Parallel()

	c := cryptoimpl.Crypto{}
	blob, err := c.EncryptStruct(mustKey(t), loginPayload{V: 1, Username: "a"})
	require.NoError(t, err)

	var out loginPayload
	err = c.DecryptStruct(mustKey(t), blob, &out)
	require.Error(t, err, "decrypt with wrong key must fail, not panic")
}

func TestDecryptStruct_TamperedBlobFails(t *testing.T) {
	t.Parallel()

	c := cryptoimpl.Crypto{}
	key := mustKey(t)
	blob, err := c.EncryptStruct(key, loginPayload{V: 1, Username: "a"})
	require.NoError(t, err)

	blob[len(blob)-1] ^= 0xff

	var out loginPayload
	require.Error(t, c.DecryptStruct(key, blob, &out))
}
