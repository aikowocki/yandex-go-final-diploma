package secret

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

// CreateSecret создаёт секрет в папке после проверки, что принадлежит пользователю.
func (u *UseCase) CreateSecret(ctx context.Context, params CreateSecretParams) (string, error) {
	if params.UserID == "" {
		return "", ErrEmptyUserID
	}
	if params.VaultID == "" {
		return "", ErrEmptyVaultID
	}
	if len(params.EncRow) == 0 {
		return "", ErrEmptyEncRow
	}
	if len(params.EncIndex) == 0 {
		return "", ErrEmptyEncIndex
	}

	owner, err := u.vaults.IsOwner(ctx, params.VaultID, params.UserID)
	if err != nil {
		return "", err
	}
	if !owner {
		return "", ErrVaultNotFound
	}

	created, err := u.secrets.Create(ctx, domain.Secret{
		VaultID:    params.VaultID,
		Type:       params.Type,
		EncRow:     params.EncRow,
		EncIndex:   params.EncIndex,
		EncPayload: params.EncPayload,
	})
	if err != nil {
		return "", err
	}
	return created.ID, nil
}
