package auth

import "errors"

var (
	ErrEncryptionNotSetup = errors.New("auth: encryption is not set up for this account")
	ErrEmptyLogin         = errors.New("auth: login must not be empty")
	ErrEmptyCredential    = errors.New("auth: login credential must not be empty")
	ErrEmptyPassphrase    = errors.New("auth: encryption passphrase must not be empty")
)
