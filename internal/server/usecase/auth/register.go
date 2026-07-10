package auth

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/alexedwards/argon2id"
)

func (u *UseCase) Register(ctx context.Context, params RegisterParams) (RegisterResult, error) {
	if params.Login == "" {
		return RegisterResult{}, ErrEmptyLogin
	}
	if len(params.LoginCredential) == 0 {
		return RegisterResult{}, ErrEmptyPassword
	}

	hash, err := argon2id.CreateHash(string(params.LoginCredential), argon2id.DefaultParams)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("hash login credential: %w", err)
	}

	user, err := u.users.Create(ctx,
		domain.User{Login: params.Login, AuthHash: hash})

	if err != nil {
		return RegisterResult{}, err
	}

	access, refresh, err := u.tokens.Issue(user.ID)
	if err != nil {
		return RegisterResult{}, err
	}

	return RegisterResult{UserID: user.ID, AccessToken: access, RefreshToken: refresh}, nil
}
