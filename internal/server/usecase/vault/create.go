package vault

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

// CreateVault создаёт папку для пользователя.
func (u *UseCase) CreateVault(ctx context.Context, params CreateParams) (string, error) {
	if params.UserID == "" {
		return "", ErrEmptyUserID
	}
	if len(params.WrappedVaultKey) == 0 {
		return "", ErrEmptyVaultKey
	}
	if len(params.EncName) == 0 {
		return "", ErrEmptyEncName
	}

	created, err := u.vaults.Create(ctx, domain.Vault{
		UserID:          params.UserID,
		WrappedVaultKey: params.WrappedVaultKey,
		EncName:         params.EncName,
	})
	if err != nil {
		return "", err
	}
	return created.ID, nil
}
