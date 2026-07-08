package auth

import (
	"context"
	"errors"
)

// EnrollTOTPParams / EnrollTOTPResult
type EnrollTOTPParams struct {
	UserID string
}

type EnrollTOTPResult struct {
	Secret string
}

var ErrNotImplemented = errors.New("auth: not implemented")

func (u *UseCase) EnrollTOTP(ctx context.Context, params EnrollTOTPParams) (EnrollTOTPResult, error) {
	// TODO
	return EnrollTOTPResult{}, ErrNotImplemented
}
