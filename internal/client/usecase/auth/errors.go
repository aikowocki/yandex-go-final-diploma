package auth

import "errors"

var (
	// ErrEncryptionNotSetup сообщает, что для аккаунта ещё не настроено шифрование.
	ErrEncryptionNotSetup = errors.New("auth: encryption is not set up for this account")
	// ErrEmptyLogin сообщает, что login не должен быть пустым.
	ErrEmptyLogin = errors.New("auth: login must not be empty")
	// ErrEmptyCredential сообщает, что учётные данные для входа не должны быть пустыми.
	ErrEmptyCredential = errors.New("auth: login credential must not be empty")
	// ErrEmptyPassphrase сообщает, что passphrase для шифрования не должен быть пустым.
	ErrEmptyPassphrase = errors.New("auth: encryption passphrase must not be empty")
	// ErrLocked сообщает, что сессия заблокирована и требует разблокировки.
	ErrLocked = errors.New("auth: session is locked (unlock first)")
)
