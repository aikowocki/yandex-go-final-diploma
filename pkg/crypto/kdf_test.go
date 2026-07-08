package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func TestParamsValidate(t *testing.T) {
	t.Parallel()

	valid := crypto.DefaultParams()

	tests := []struct {
		name    string
		params  crypto.Params
		wantErr bool
	}{
		{
			name:    "default params are valid",
			params:  valid,
			wantErr: false,
		},
		{
			name: "unsupported version",
			params: crypto.Params{
				Version:     999,
				Memory:      valid.Memory,
				Iterations:  valid.Iterations,
				Parallelism: valid.Parallelism,
				KeyLen:      valid.KeyLen,
			},
			wantErr: true,
		},
		{
			name: "zero memory",
			params: crypto.Params{
				Version:     crypto.ParamsVersionV1,
				Memory:      0,
				Iterations:  valid.Iterations,
				Parallelism: valid.Parallelism,
				KeyLen:      valid.KeyLen,
			},
			wantErr: true,
		},
		{
			name: "zero iterations",
			params: crypto.Params{
				Version:     crypto.ParamsVersionV1,
				Memory:      valid.Memory,
				Iterations:  0,
				Parallelism: valid.Parallelism,
				KeyLen:      valid.KeyLen,
			},
			wantErr: true,
		},
		{
			name: "zero parallelism",
			params: crypto.Params{
				Version:     crypto.ParamsVersionV1,
				Memory:      valid.Memory,
				Iterations:  valid.Iterations,
				Parallelism: 0,
				KeyLen:      valid.KeyLen,
			},
			wantErr: true,
		},
		{
			name: "key length too short",
			params: crypto.Params{
				Version:     crypto.ParamsVersionV1,
				Memory:      valid.Memory,
				Iterations:  valid.Iterations,
				Parallelism: valid.Parallelism,
				KeyLen:      8,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.params.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateSalt(t *testing.T) {
	t.Parallel()

	salt1, err := crypto.GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt1, crypto.SaltLen)

	salt2, err := crypto.GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt2, crypto.SaltLen)

	assert.NotEqual(t, salt1, salt2, "two generated salts must not be equal")
}

func TestArgon2id_Deterministic(t *testing.T) {
	t.Parallel()

	params := crypto.DefaultParams()
	secret := []byte("correct-horse-battery-staple")
	salt := []byte("0123456789abcdef")

	seed1, err := crypto.Argon2id(secret, salt, params)
	require.NoError(t, err)

	seed2, err := crypto.Argon2id(secret, salt, params)
	require.NoError(t, err)

	assert.Equal(t, seed1, seed2, "same secret+salt+params must produce the same seed")
	assert.Len(t, seed1, int(params.KeyLen))
}

func TestArgon2id_DifferentSaltProducesDifferentSeed(t *testing.T) {
	t.Parallel()

	params := crypto.DefaultParams()
	secret := []byte("correct-horse-battery-staple")

	seed1, err := crypto.Argon2id(secret, []byte("0123456789abcdef"), params)
	require.NoError(t, err)

	seed2, err := crypto.Argon2id(secret, []byte("fedcba9876543210"), params)
	require.NoError(t, err)

	assert.NotEqual(t, seed1, seed2)
}

func TestArgon2id_DifferentSecretProducesDifferentSeed(t *testing.T) {
	t.Parallel()

	params := crypto.DefaultParams()
	salt := []byte("0123456789abcdef")

	seed1, err := crypto.Argon2id([]byte("login-credential"), salt, params)
	require.NoError(t, err)

	seed2, err := crypto.Argon2id([]byte("encryption-passphrase"), salt, params)
	require.NoError(t, err)

	assert.NotEqual(t, seed1, seed2)
}

func TestArgon2id_Errors(t *testing.T) {
	t.Parallel()

	params := crypto.DefaultParams()
	salt := []byte("0123456789abcdef")
	secret := []byte("secret")

	t.Run("empty secret", func(t *testing.T) {
		t.Parallel()
		_, err := crypto.Argon2id(nil, salt, params)
		assert.Error(t, err)
	})

	t.Run("empty salt", func(t *testing.T) {
		t.Parallel()
		_, err := crypto.Argon2id(secret, nil, params)
		assert.Error(t, err)
	})

	t.Run("invalid params", func(t *testing.T) {
		t.Parallel()
		invalid := params
		invalid.Memory = 0
		_, err := crypto.Argon2id(secret, salt, invalid)
		assert.Error(t, err)
	})
}

// TestDifferentSecretsProduceUnrelatedDerivedKeys проверяет сквозную цепочку
// Argon2id → HKDF: seed'ы и производные ключи, выведенные из разных секретов,
// не совпадают даже при случайно одинаковой salt.
func TestDifferentSecretsProduceUnrelatedDerivedKeys(t *testing.T) {
	t.Parallel()

	params := crypto.DefaultParams()
	sharedSalt := []byte("0123456789abcdef")

	seedA, err := crypto.Argon2id([]byte("secret-a"), sharedSalt, params)
	require.NoError(t, err)

	seedB, err := crypto.Argon2id([]byte("secret-b"), sharedSalt, params)
	require.NoError(t, err)

	assert.NotEqual(t, seedA, seedB, "seeds from different secrets must differ even with the same salt")

	keyA, err := crypto.HKDF(seedA, crypto.InfoEncryption, 32)
	require.NoError(t, err)

	keyB, err := crypto.HKDF(seedB, crypto.InfoEncryption, 32)
	require.NoError(t, err)

	assert.NotEqual(t, keyA, keyB)
	assert.NotEqual(t, seedA, keyA)
	assert.NotEqual(t, seedB, keyB)
}
