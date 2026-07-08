package auth

import "errors"

var (
	ErrLoginTaken = errors.New("auth: login already taken")

	ErrInvalidCredentials = errors.New("auth: invalid login or password")

	ErrUserNotFound = errors.New("auth: user not found")

	ErrInvalidRefreshToken = errors.New("auth: invalid refresh token")

	ErrEmptyLogin = errors.New("auth: login must not be empty")

	ErrEmptyPassword = errors.New("auth: password must not be empty")
)
