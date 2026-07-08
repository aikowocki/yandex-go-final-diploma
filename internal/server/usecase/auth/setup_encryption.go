package auth

import (
	"context"
	"errors"
	"fmt"
)

func (u *UseCase) SetupEncryption(ctx context.Context, params SetupEncryptionParams) error {
	_, err := u.users.GetByID(ctx, params.UserID)
	if errors.Is(err, ErrUserNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("get user by id: %w", err)
	}

	if err := u.users.UpdateEncKDF(ctx, params.UserID, params.EncKDFSalt, params.EncKDFParams); err != nil {
		return fmt.Errorf("update enc kdf: %w", err)
	}

	return nil
}
