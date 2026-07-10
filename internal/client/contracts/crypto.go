package contracts

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// Crypto — вывод ключей (KDF).
type Crypto interface {
	DeriveMasterSeed(key domain.EncryptionPassphrase, salt []byte, params crypto.Params) ([]byte, error)
	DeriveMasterKey(masterSeed []byte) ([]byte, error)
}

// Cipher — симметричное шифрование данных и envelope-обёртка ключей.
type Cipher interface {
	GenerateVaultKey() ([]byte, error)
	WrapVaultKey(vaultKey, masterKey []byte) (wrapped []byte, err error)
	UnwrapVaultKey(wrapped, masterKey []byte) (vaultKey []byte, err error)
	EncryptStruct(key, ad []byte, value any) ([]byte, error)
	DecryptStruct(key, ad, blob []byte, value any) error
}
