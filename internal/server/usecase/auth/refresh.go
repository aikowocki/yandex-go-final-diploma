package auth

import (
	"context"
	"errors"
	"fmt"
)

// RefreshToken обновляет пару access+refresh JWT по действующему refresh-токену.
func (u *UseCase) RefreshToken(ctx context.Context, params RefreshParams) (AuthResult, error) {
	userID, err := u.tokens.Verify(params.RefreshToken)
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

	access, refresh, err := u.tokens.Refresh(params.RefreshToken)
	if err != nil {
		return AuthResult{}, ErrInvalidRefreshToken
	}

	return AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		EncKDFSalt:   user.EncKDFSalt,
		EncKDFParams: user.EncKDFParams,
	}, nil
}
