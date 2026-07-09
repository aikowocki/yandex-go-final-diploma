package cryptoimpl

import (
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// EncryptStruct сериализует value в JSON и шифрует AEAD под key.
func (c Crypto) EncryptStruct(key []byte, value any) ([]byte, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("cryptoimpl: marshal struct: %w", err)
	}
	return crypto.Encrypt(key, plaintext)
}

// DecryptStruct расшифровывает блоб под key и десериализует JSON в value.
func (c Crypto) DecryptStruct(key, blob []byte, value any) error {
	plaintext, err := crypto.Decrypt(key, blob)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(plaintext, value); err != nil {
		return fmt.Errorf("cryptoimpl: unmarshal struct: %w", err)
	}
	return nil
}
