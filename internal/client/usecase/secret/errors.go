package secret

import "errors"

var (
	// ErrVaultLocked — папка не открыта в сессии (нет VaultKey): сначала нужен vault list/open.
	ErrVaultLocked         = errors.New("secret: vault is locked (open the vault first)")
	ErrEmptyVaultID        = errors.New("secret: vault id must not be empty")
	ErrEmptySecretID       = errors.New("secret: secret id must not be empty")
	ErrEmptyTitle          = errors.New("secret: title must not be empty")
	ErrNilConflict         = errors.New("secret: nil conflict")
	ErrUnknownChoice       = errors.New("secret: unknown conflict choice")
	ErrIndexTooLarge       = errors.New("secret: index (note/custom fields) is too large, create a separate text secret instead")
	ErrEmptyTOTPSecret     = errors.New("secret: totp secret must not be empty")
	ErrInvalidOTPAuthURI   = errors.New("secret: invalid otpauth:// uri")
	ErrEmptyBinaryData     = errors.New("secret: binary data source must not be empty")
	ErrInvalidOTPCodeIndex = errors.New("secret: invalid OTP code index")
	ErrNoOTPCodes          = errors.New("secret: secret has no recovery codes")
	ErrSecretNotFound      = errors.New("secret: not found in local cache (run sync first)")
)
