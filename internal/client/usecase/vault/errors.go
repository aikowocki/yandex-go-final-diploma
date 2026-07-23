package vault

import "errors"

var (
	// ErrLocked — сессия заблокирована (MasterKey не выведен): сначала нужен login+unlock.
	ErrLocked = errors.New("vault: session is locked (unlock required)")
	// ErrEmptyName — имя vault'а не может быть пустым.
	ErrEmptyName = errors.New("vault: name must not be empty")
)
