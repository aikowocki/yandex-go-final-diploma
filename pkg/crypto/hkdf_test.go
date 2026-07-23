package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func TestHKDF_Deterministic(t *testing.T) {
	t.Parallel()

	seed := []byte("some-derived-seed-material")

	key1, err := crypto.HKDF(seed, crypto.InfoEncryption, 32)
	require.NoError(t, err)

	key2, err := crypto.HKDF(seed, crypto.InfoEncryption, 32)
	require.NoError(t, err)

	assert.Equal(t, key1, key2, "same seed+info+keyLen must produce the same key")
	assert.Len(t, key1, 32)
}

func TestHKDF_DifferentInfoProducesDifferentKey(t *testing.T) {
	t.Parallel()

	seed := []byte("some-derived-seed-material")

	key1, err := crypto.HKDF(seed, crypto.InfoEncryption, 32)
	require.NoError(t, err)

	key2, err := crypto.HKDF(seed, "some-other-info", 32)
	require.NoError(t, err)

	assert.NotEqual(t, key1, key2)
}

func TestHKDF_DifferentSeedProducesDifferentKey(t *testing.T) {
	t.Parallel()

	key1, err := crypto.HKDF([]byte("seed-one"), crypto.InfoEncryption, 32)
	require.NoError(t, err)

	key2, err := crypto.HKDF([]byte("seed-two"), crypto.InfoEncryption, 32)
	require.NoError(t, err)

	assert.NotEqual(t, key1, key2)
}

func TestHKDF_Errors(t *testing.T) {
	t.Parallel()

	seed := []byte("some-derived-seed-material")

	t.Run("empty seed", func(t *testing.T) {
		t.Parallel()
		_, err := crypto.HKDF(nil, crypto.InfoEncryption, 32)
		assert.Error(t, err)
	})

	t.Run("empty info", func(t *testing.T) {
		t.Parallel()
		_, err := crypto.HKDF(seed, "", 32)
		assert.Error(t, err)
	})

	t.Run("non-positive key length", func(t *testing.T) {
		t.Parallel()
		_, err := crypto.HKDF(seed, crypto.InfoEncryption, 0)
		assert.Error(t, err)
	})
}
