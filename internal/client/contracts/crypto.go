package contracts

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

type Crypto interface {
	DeriveMasterSeed(key domain.EncryptionPassphrase, salt []byte, params crypto.Params) ([]byte, error)
	DeriveMasterKey(masterSeed []byte) ([]byte, error)
}
