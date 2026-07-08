package auth

import (
	"context"
	"fmt"
)

// Register регистрирует пользователя на сервере и сохраняет полученные токены локально.
func (u *UseCase) Register(ctx context.Context, login string, loginCredential []byte) error {
	if login == "" {
		return ErrEmptyLogin
	}
	if len(loginCredential) == 0 {
		return ErrEmptyCredential
	}

	tokens, err := u.server.Register(ctx, login, loginCredential)
	if err != nil {
		return err
	}

	if err := u.tokens.Save(tokens); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}

	return nil
}
