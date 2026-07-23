package cryptoimpl

import "github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"

const derivedKeyLen = 32

// DeriveMasterKey раскрывает MasterSeed через HKDF в MasterKey фиксированной длины.
func (c Crypto) DeriveMasterKey(masterSeed []byte) ([]byte, error) {
	return crypto.HKDF(masterSeed, crypto.InfoEncryption, derivedKeyLen)
}
