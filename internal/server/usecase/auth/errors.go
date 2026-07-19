package auth

import "errors"

var (
	// ErrLoginTaken — логин уже занят другим пользователем.
	ErrLoginTaken = errors.New("auth: login already taken")

	// ErrInvalidCredentials — неверный логин или пароль.
	ErrInvalidCredentials = errors.New("auth: invalid login or password")

	// ErrUserNotFound — пользователь не найден.
	ErrUserNotFound = errors.New("auth: user not found")

	// ErrInvalidRefreshToken — refresh-токен недействителен или просрочен.
	ErrInvalidRefreshToken = errors.New("auth: invalid refresh token")

	// ErrEmptyLogin — логин не может быть пустым.
	ErrEmptyLogin = errors.New("auth: login must not be empty")

	// ErrEmptyPassword — пароль не может быть пустым.
	ErrEmptyPassword = errors.New("auth: password must not be empty")

	// ErrEmptyUserID — id пользователя не может быть пустым.
	ErrEmptyUserID = errors.New("auth: user id must not be empty")
)
