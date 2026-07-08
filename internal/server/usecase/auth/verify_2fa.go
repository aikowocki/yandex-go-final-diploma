package auth

import "context"

// VerifyTOTPParams — заглушка. TODO 2fa
type VerifyTOTPParams struct {
	UserID string
	Code   string
}

func (u *UseCase) VerifyTOTP(ctx context.Context, params VerifyTOTPParams) error {
	return ErrNotImplemented
}
