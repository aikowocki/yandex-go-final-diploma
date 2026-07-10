package secret

import "errors"

var (
	// ErrVaultLocked — папка не открыта в сессии (нет VaultKey): сначала нужен vault list/open.
	ErrVaultLocked   = errors.New("secret: vault is locked (open the vault first)")
	ErrEmptyVaultID  = errors.New("secret: vault id must not be empty")
	ErrEmptySecretID = errors.New("secret: secret id must not be empty")
	ErrEmptyTitle    = errors.New("secret: title must not be empty")
	ErrNilConflict   = errors.New("secret: nil conflict")
	ErrUnknownChoice = errors.New("secret: unknown conflict choice")
)
