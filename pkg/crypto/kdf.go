package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// ParamsVersionV1 — первая версия набора параметров Argon2id.
// Хранится вместе с параметрами (в *_kdf_params), чтобы при их изменении
// в будущем можно было различать, какой версией были выведены существующие ключи.
const ParamsVersionV1 = 1

// SaltLen — длина соли в байтах для Argon2id.
const SaltLen = 16

// Params описывает параметры Argon2id.
// Хранится как JSON в колонке enc_kdf_params (клиентский вывод MasterSeed из EncryptionPassphrase).
// Для логина клиентского KDF нет — параметры серверного Argon2id живут внутри PHC-строки auth_hash.
type Params struct {
	Version     int    `json:"version"`
	Memory      uint32 `json:"memory_kib"`  // Память, KiB
	Iterations  uint32 `json:"iterations"`  // Число итераций (time cost)
	Parallelism uint8  `json:"parallelism"` // Число потоков
	KeyLen      uint32 `json:"key_len"`     // Длина выходного ключа (seed), байты
}

// DefaultParams возвращает набор параметров Argon2id с запасом над базовой
// рекомендацией OWASP 2026 (m=19 MiB, t=2, p=1). TODO провести бенчмарки?
// | Memory | Iterations | Parallelism |
// | -----: | ---------: | ----------: |
// | 46 MiB |          1 |           1 |
// | 19 MiB |          2 |           1 |
// | 12 MiB |          3 |           1 |
// |  9 MiB |          4 |           1 |
// |  7 MiB |          5 |           1 |
func DefaultParams() Params {
	return Params{
		Version:     ParamsVersionV1,
		Memory:      64 * 1024, // 64 MiB
		Iterations:  3,
		Parallelism: 2,
		KeyLen:      32,
	}
}

// Validate проверяет параметры.
func (p Params) Validate() error {
	switch {
	case p.Version != ParamsVersionV1:
		return fmt.Errorf("crypto: unsupported params version %d", p.Version)
	case p.Memory == 0:
		return errors.New("crypto: memory must be > 0")
	case p.Iterations == 0:
		return errors.New("crypto: iterations must be > 0")
	case p.Parallelism == 0:
		return errors.New("crypto: parallelism must be > 0")
	case p.KeyLen < 16:
		return errors.New("crypto: key length must be >= 16 bytes")
	default:
		return nil
	}
}

// GenerateSalt возвращает криптографически случайную соль длины SaltLen.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("crypto: generate salt: %w", err)
	}
	return salt, nil
}

// Argon2id выводит seed из secret и salt по заданным params.
func Argon2id(secret, salt []byte, params Params) ([]byte, error) {
	if len(secret) == 0 {
		return nil, errors.New("crypto: secret must not be empty")
	}
	if len(salt) == 0 {
		return nil, errors.New("crypto: salt must not be empty")
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}

	seed := argon2.IDKey(secret, salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLen)
	return seed, nil
}
