package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexedwards/argon2id"
)

func (u *UseCase) Login(ctx context.Context, params LoginParams) (AuthResult, error) {
	if params.Login == "" {
		return AuthResult{}, ErrEmptyLogin
	}
	if len(params.LoginCredential) == 0 {
		return AuthResult{}, ErrEmptyPassword
	}

	user, err := u.users.GetByLogin(ctx, params.Login)
	if errors.Is(err, ErrUserNotFound) {
		return AuthResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, err
	}

	match, err := argon2id.ComparePasswordAndHash(string(params.LoginCredential), user.AuthHash)
	if err != nil {
		return AuthResult{}, fmt.Errorf("compare password and hash: %w", err)
	}
	if !match {
		return AuthResult{}, ErrInvalidCredentials
	}

	access, refresh, err := u.tokens.Issue(user.ID)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		EncKDFSalt:   user.EncKDFSalt,
		EncKDFParams: user.EncKDFParams,
	}, nil
}
