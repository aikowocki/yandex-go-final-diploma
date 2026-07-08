package cryptoimpl

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

var _ contracts.Crypto = Crypto{}

type Crypto struct {
}

func (c Crypto) DeriveMasterSeed(key domain.EncryptionPassphrase, salt []byte, params crypto.Params) ([]byte, error) {
	return crypto.Argon2id(key, salt, params)
}
