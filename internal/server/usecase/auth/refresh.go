package auth

import (
	"context"
	"errors"
	"fmt"
)

// RefreshToken обновляет пару access+refresh JWT по действующему refresh-токену.
func (u *UseCase) RefreshToken(ctx context.Context, params RefreshParams) (Result, error) {
	userID, err := u.tokens.VerifyRefresh(params.RefreshToken)
	if err != nil {
		return Result{}, ErrInvalidRefreshToken
	}

	user, err := u.users.GetByID(ctx, userID)
	if errors.Is(err, ErrUserNotFound) {
		return Result{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return Result{}, fmt.Errorf("get user by id: %w", err)
	}

	// userID уже получен из refresh-токена — выпускаем новую пару напрямую (без повторного парсинга).
	access, refresh, err := u.tokens.Issue(userID)
	if err != nil {
		return Result{}, fmt.Errorf("issue tokens: %w", err)
	}

	return Result{
		UserID:       userID,
		AccessToken:  access,
		RefreshToken: refresh,
		EncKDFSalt:   user.EncKDFSalt,
		EncKDFParams: user.EncKDFParams,
		EncMasterKey: user.EncMasterKey,
	}, nil
}
