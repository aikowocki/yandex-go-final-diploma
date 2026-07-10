package auth

import (
	"context"
	"errors"
	"fmt"
)

// RefreshToken обновляет пару access+refresh JWT по действующему refresh-токену.
func (u *UseCase) RefreshToken(ctx context.Context, params RefreshParams) (AuthResult, error) {
	userID, err := u.tokens.VerifyRefresh(params.RefreshToken)
	if err != nil {
		return AuthResult{}, ErrInvalidRefreshToken
	}

	user, err := u.users.GetByID(ctx, userID)
	if errors.Is(err, ErrUserNotFound) {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return AuthResult{}, fmt.Errorf("get user by id: %w", err)
	}

	// userID уже получен из refresh-токена — выпускаем новую пару напрямую (без повторного парсинга).
	access, refresh, err := u.tokens.Issue(userID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("issue tokens: %w", err)
	}

	return AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		EncKDFSalt:   user.EncKDFSalt,
		EncKDFParams: user.EncKDFParams,
	}, nil
}
