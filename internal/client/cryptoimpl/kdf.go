package cryptoimpl

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

var _ contracts.Crypto = Crypto{}

// Crypto — реализация contracts.Crypto на основе Argon2id/HKDF (pkg/crypto).
type Crypto struct {
}

// DeriveMasterSeed раскрывает MasterSeed из кодовой фразы шифрования через Argon2id.
func (c Crypto) DeriveMasterSeed(key domain.EncryptionPassphrase, salt []byte, params crypto.Params) ([]byte, error) {
	return crypto.Argon2id(key, salt, params)
}
