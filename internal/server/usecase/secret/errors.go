package secret

import "errors"

var (
	ErrEmptyUserID    = errors.New("secret: user id must not be empty")
	ErrEmptyVaultID   = errors.New("secret: vault id must not be empty")
	ErrEmptySecretID  = errors.New("secret: secret id must not be empty")
	ErrEmptyEncRow    = errors.New("secret: enc_row must not be empty")
	ErrEmptyEncIndex  = errors.New("secret: enc_index must not be empty")
	ErrVaultNotFound  = errors.New("secret: vault not found")
	ErrSecretNotFound = errors.New("secret: secret not found")
)
