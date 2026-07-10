package cli

import (
	"crypto/subtle"
	"errors"
)

// errMismatch — введённые дважды секреты не совпали.
var errMismatch = errors.New("entries do not match")

// errInvalidChoice — пользователь не выбрал корректный вариант разрешения конфликта.
var errInvalidChoice = errors.New("no valid conflict choice provided")

// bytesEqual сравнивает секреты за постоянное время.
func bytesEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
