package auth

import (
	"context"
	"errors"
)

// EnrollTOTPParams содержит параметры для EnrollTOTP. // TODO
type EnrollTOTPParams struct {
	UserID string
}

// EnrollTOTPResult содержит результат выполнения EnrollTOTP. // TODO
type EnrollTOTPResult struct {
	Secret string
}

// ErrNotImplemented возвращается методами 2FA.
var ErrNotImplemented = errors.New("auth: not implemented")

// EnrollTOTP подключает для пользователя двухфакторную аутентификацию по TOTP. 	// TODO
func (u *UseCase) EnrollTOTP(ctx context.Context, params EnrollTOTPParams) (EnrollTOTPResult, error) {
	return EnrollTOTPResult{}, ErrNotImplemented
}
