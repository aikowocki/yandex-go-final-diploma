package auth

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

// Register создаёт новую учётную запись пользователя и выдаёт пару токенов access/refresh.
func (u *UseCase) Register(ctx context.Context, params RegisterParams) (RegisterResult, error) {
	if params.Login == "" {
		return RegisterResult{}, ErrEmptyLogin
	}
	if len(params.LoginCredential) == 0 {
		return RegisterResult{}, ErrEmptyPassword
	}

	hash, err := hashPassword(ctx, string(params.LoginCredential))
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
