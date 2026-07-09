package vault

import "errors"

var (
	ErrEmptyUserID   = errors.New("vault: user id must not be empty")
	ErrEmptyVaultKey = errors.New("vault: wrapped vault key must not be empty")
	ErrEmptyEncName  = errors.New("vault: encrypted name must not be empty")
)
