package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// Выбор AEAD: XChaCha20-Poly1305.
const (
	// KeySize — длина симметричного ключа AEAD, байты.
	KeySize = chacha20poly1305.KeySize
	// NonceSize — длина nonce XChaCha20, байты.
	NonceSize = chacha20poly1305.NonceSizeX
)

var (
	ErrInvalidKeySize = fmt.Errorf("crypto: key must be %d bytes", KeySize)
	// ErrCiphertextTooShort — блоб короче nonce, разобрать нельзя.
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")
)

// Encrypt шифрует plaintext ключом key (AEAD) без associated data. Возвращает
// самодостаточный блоб nonce||ciphertext (ciphertext уже включает Poly1305-тег).
// Nonce — случайный на каждый вызов, поэтому шифрование недетерминировано.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	return EncryptWithAD(key, plaintext, nil)
}

// Decrypt расшифровывает блоб nonce||ciphertext ключом key без associated data.
func Decrypt(key, blob []byte) ([]byte, error) {
	return DecryptWithAD(key, blob, nil)
}

// EncryptWithAD шифрует plaintext ключом key, привязывая шифротекст к associated data ad
// (AAD не шифруется, но включается в подпись Poly1305). Расшифровка пройдёт только при
// точном совпадении ad. Используется для anti-tampering/anti-rollback: в ad передаётся
// детерминированный контекст (например vault_id|secret_id|version). ad=nil эквивалентно
// шифрованию без контекста.
func EncryptWithAD(key, plaintext, ad []byte) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: generate nonce: %w", err)
	}

	// Seal с dst=nonce дописывает ciphertext после nonce → на выходе nonce||ciphertext.
	return aead.Seal(nonce, nonce, plaintext, ad), nil
}

// DecryptWithAD расшифровывает блоб nonce||ciphertext ключом key с проверкой associated data ad.
// Неверный ключ, повреждённый шифротекст ИЛИ несовпадающая ad дают ошибку (не панику).
func DecryptWithAD(key, blob, ad []byte) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}

	if len(blob) < NonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, ciphertext := blob[:NonceSize], blob[NonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, ad)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}
	return plaintext, nil
}

// WrapKey оборачивает (шифрует) ключ key под key-encryption-key kek — envelope encryption
// (например VaultKey под MasterKey).
func WrapKey(kek, key []byte) ([]byte, error) {
	return Encrypt(kek, key)
}

// UnwrapKey разворачивает обёрнутый WrapKey ключ.
func UnwrapKey(kek, wrapped []byte) ([]byte, error) {
	return Decrypt(kek, wrapped)
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	return chacha20poly1305.NewX(key)
}

// GenerateKey возвращает криптографически случайный симметричный ключ длины KeySize
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("crypto: generate key: %w", err)
	}
	return key, nil
}
