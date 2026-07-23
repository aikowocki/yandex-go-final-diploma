package secret

import "errors"

var (
	// ErrEmptyUserID — user id не передан.
	ErrEmptyUserID = errors.New("secret: user id must not be empty")
	// ErrEmptyVaultID — vault id не передан.
	ErrEmptyVaultID = errors.New("secret: vault id must not be empty")
	// ErrEmptySecretID — secret id не передан.
	ErrEmptySecretID = errors.New("secret: secret id must not be empty")
	// ErrEmptyEncRow — enc_row не передан.
	ErrEmptyEncRow = errors.New("secret: enc_row must not be empty")
	// ErrEmptyEncIndex — enc_index не передан.
	ErrEmptyEncIndex = errors.New("secret: enc_index must not be empty")
	// ErrVaultNotFound — папка не найдена.
	ErrVaultNotFound = errors.New("secret: vault not found")
	// ErrSecretNotFound — секрет не найден.
	ErrSecretNotFound = errors.New("secret: secret not found")
	// ErrEmptyBlobRef — blob ref не передан.
	ErrEmptyBlobRef = errors.New("secret: blob ref must not be empty")
	// ErrNotBinarySecret — AttachBlob вызван для секрета не типа binary.
	ErrNotBinarySecret = errors.New("secret: AttachBlob is only valid for type=binary secrets")
)
