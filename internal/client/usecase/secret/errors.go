package secret

import "errors"

var (
	// ErrVaultLocked — папка не открыта в сессии (нет VaultKey): сначала нужен vault list/open.
	ErrVaultLocked = errors.New("secret: vault is locked (open the vault first)")
	// ErrEmptyVaultID сообщает, что id папки не должен быть пустым.
	ErrEmptyVaultID = errors.New("secret: vault id must not be empty")
	// ErrEmptySecretID сообщает, что id секрета не должен быть пустым.
	ErrEmptySecretID = errors.New("secret: secret id must not be empty")
	// ErrEmptyTitle сообщает, что заголовок секрета не должен быть пустым.
	ErrEmptyTitle = errors.New("secret: title must not be empty")
	// ErrNilConflict сообщает, что передан nil-конфликт.
	ErrNilConflict = errors.New("secret: nil conflict")
	// ErrUnknownChoice сообщает о неизвестном варианте разрешения конфликта.
	ErrUnknownChoice = errors.New("secret: unknown conflict choice")
	// ErrIndexTooLarge сообщает, что index-тир (note/custom fields) слишком велик.
	ErrIndexTooLarge = errors.New("secret: index (note/custom fields) is too large, create a separate text secret instead")
	// ErrEmptyTOTPSecret сообщает, что секрет TOTP не должен быть пустым.
	ErrEmptyTOTPSecret = errors.New("secret: totp secret must not be empty")
	// ErrInvalidOTPAuthURI сообщает о некорректном otpauth:// URI.
	ErrInvalidOTPAuthURI = errors.New("secret: invalid otpauth:// uri")
	// ErrEmptyBinaryData сообщает, что источник бинарных данных не должен быть пустым.
	ErrEmptyBinaryData = errors.New("secret: binary data source must not be empty")
	// ErrInvalidOTPCodeIndex сообщает о некорректном индексе OTP-кода.
	ErrInvalidOTPCodeIndex = errors.New("secret: invalid OTP code index")
	// ErrNoOTPCodes сообщает, что у секрета нет резервных кодов.
	ErrNoOTPCodes = errors.New("secret: secret has no recovery codes")
	// ErrSecretNotFound сообщает, что секрет не найден в локальном кеше.
	ErrSecretNotFound = errors.New("secret: not found in local cache (run sync first)")
	// ErrOutboxEntryNotFound — outbox-запись не найдена, либо не относится к секрету, либо
	// её операция (create/blob_upload) не может участвовать в конфликте версий.
	ErrOutboxEntryNotFound = errors.New("secret: outbox entry not found or not a version conflict")
)
