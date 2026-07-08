package crypto

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// InfoEncryption — контекстная метка (info) для HKDF при выводе MasterKey из MasterSeed
const InfoEncryption = "encryption"

// HKDF выводит derivedKey длины keyLen из seed, используя info.
// seed — это выход Argon2id (MasterSeed).
func HKDF(seed []byte, info string, keyLen int) ([]byte, error) {
	if len(seed) == 0 {
		return nil, errors.New("crypto: seed must not be empty")
	}
	if info == "" {
		return nil, errors.New("crypto: info must not be empty")
	}
	if keyLen <= 0 {
		return nil, errors.New("crypto: keyLen must be > 0")
	}

	reader := hkdf.New(sha256.New, seed, nil, []byte(info))
	derived := make([]byte, keyLen)
	if _, err := io.ReadFull(reader, derived); err != nil {
		return nil, fmt.Errorf("crypto: hkdf expand: %w", err)
	}
	return derived, nil
}
