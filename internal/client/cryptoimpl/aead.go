package cryptoimpl

import (
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// EncryptStruct сериализует value в JSON и шифрует AEAD под key, привязывая шифротекст
// к associated data ad (не шифруется, но входит в подпись). ad=nil — без контекста.
func (c Crypto) EncryptStruct(key, ad []byte, value any) ([]byte, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("cryptoimpl: marshal struct: %w", err)
	}
	return crypto.EncryptWithAD(key, plaintext, ad)
}

// DecryptStruct расшифровывает блоб под key с проверкой associated data ad и десериализует JSON в value.
func (c Crypto) DecryptStruct(key, ad, blob []byte, value any) error {
	plaintext, err := crypto.DecryptWithAD(key, blob, ad)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(plaintext, value); err != nil {
		return fmt.Errorf("cryptoimpl: unmarshal struct: %w", err)
	}
	return nil
}
