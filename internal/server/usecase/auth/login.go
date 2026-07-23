package auth

import (
	"context"
	"errors"
	"fmt"
)

// Login проверяет учётные данные пользователя и выдаёт новую пару токенов access/refresh.
func (u *UseCase) Login(ctx context.Context, params LoginParams) (Result, error) {
	if params.Login == "" {
		return Result{}, ErrEmptyLogin
	}
	if len(params.LoginCredential) == 0 {
		return Result{}, ErrEmptyPassword
	}

	user, err := u.users.GetByLogin(ctx, params.Login)
	if errors.Is(err, ErrUserNotFound) {
		return Result{}, ErrInvalidCredentials
	}
	if err != nil {
		return Result{}, err
	}

	match, err := comparePassword(ctx, string(params.LoginCredential), user.AuthHash)
	if err != nil {
		return Result{}, fmt.Errorf("compare password and hash: %w", err)
	}
	if !match {
		return Result{}, ErrInvalidCredentials
	}

	access, refresh, err := u.tokens.Issue(user.ID)
	if err != nil {
		return Result{}, err
	}

	return Result{
		UserID:       user.ID,
		AccessToken:  access,
		RefreshToken: refresh,
		EncKDFSalt:   user.EncKDFSalt,
		EncKDFParams: user.EncKDFParams,
		EncMasterKey: user.EncMasterKey,
	}, nil
}
