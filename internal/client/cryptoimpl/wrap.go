package cryptoimpl

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

var _ contracts.Cipher = Crypto{}

// GenerateVaultKey генерирует случайный VaultKey (DEK) для новой папки.
func (c Crypto) GenerateVaultKey() ([]byte, error) {
	return crypto.GenerateKey()
}

// WrapVaultKey оборачивает VaultKey под MasterKey (envelope encryption).
// MasterKey выступает key-encryption-key. На сервер уходит только wrapped.
func (c Crypto) WrapVaultKey(vaultKey, masterKey []byte) ([]byte, error) {
	return crypto.WrapKey(masterKey, vaultKey)
}

// UnwrapVaultKey разворачивает VaultKey из wrapped под MasterKey.
func (c Crypto) UnwrapVaultKey(wrapped, masterKey []byte) ([]byte, error) {
	return crypto.UnwrapKey(masterKey, wrapped)
}
