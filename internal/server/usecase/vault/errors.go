package vault

import "errors"

var (
	// ErrEmptyUserID — user id не передан.
	ErrEmptyUserID = errors.New("vault: user id must not be empty")
	// ErrEmptyVaultKey — wrapped vault key не передан.
	ErrEmptyVaultKey = errors.New("vault: wrapped vault key must not be empty")
	// ErrEmptyEncName — зашифрованное имя папки не передано.
	ErrEmptyEncName = errors.New("vault: encrypted name must not be empty")
)
