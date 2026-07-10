package vault

import "errors"

var (
	// ErrLocked — сессия заблокирована (MasterKey не выведен): сначала нужен login+unlock.
	ErrLocked    = errors.New("vault: session is locked (unlock required)")
	ErrEmptyName = errors.New("vault: name must not be empty")
)
