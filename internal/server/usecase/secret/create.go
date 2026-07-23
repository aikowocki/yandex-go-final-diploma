package secret

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

// CreateSecret создаёт секрет в папке после проверки, что она принадлежит пользователю.
// id секрета приходит от клиента (нужен для AAD). Всё делается в одной транзакции:
// проверка владения + вставка + бамп версии папки (сигнал sync другим устройствам).
func (u *UseCase) CreateSecret(ctx context.Context, params CreateParams) (string, error) {
	if params.UserID == "" {
		return "", ErrEmptyUserID
	}
	if params.VaultID == "" {
		return "", ErrEmptyVaultID
	}
	if params.SecretID == "" {
		return "", ErrEmptySecretID
	}
	if len(params.EncRow) == 0 {
		return "", ErrEmptyEncRow
	}
	if len(params.EncIndex) == 0 {
		return "", ErrEmptyEncIndex
	}

	var createdID string
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		owner, err := u.vaults.IsOwner(ctx, params.VaultID, params.UserID)
		if err != nil {
			return err
		}
		if !owner {
			return ErrVaultNotFound
		}

		created, err := u.secrets.Create(ctx, domain.Secret{
			ID:         params.SecretID,
			VaultID:    params.VaultID,
			Type:       params.Type,
			EncRow:     params.EncRow,
			EncIndex:   params.EncIndex,
			EncPayload: params.EncPayload,
		})
		if err != nil {
			return err
		}
		createdID = created.ID
		return u.secrets.BumpVaultVersion(ctx, params.VaultID)
	})
	if err != nil {
		return "", err
	}
	return createdID, nil
}
