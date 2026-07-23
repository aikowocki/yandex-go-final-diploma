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

	if err := u.reconcileAccount(ctx, tokens.UserID); err != nil {
		return err
	}

	// Кешируем login для отображения в UI (Lock-экран, User-меню), как в Login.
	_ = u.local.KVSet(ctx, kvAccountLogin, []byte(login))

	if err := u.tokens.Save(tokens); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}

	return nil
}
