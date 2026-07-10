package secret

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// GetPayload запрашивает Tier 3-тело секрета и расшифровывает его VaultKey'ом ваулта.
// vaultID нужен, чтобы выбрать VaultKey.
func (u *UseCase) GetPayload(ctx context.Context, vaultID, secretID string) (DecryptedPayload, error) {
	if secretID == "" {
		return DecryptedPayload{}, ErrEmptySecretID
	}

	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return DecryptedPayload{}, err
	}

	item, err := u.server.GetSecretPayload(ctx, token, secretID)
	if err != nil {
		return DecryptedPayload{}, err
	}

	var payload secretcontent.LoginPasswordPayload
	if err := u.cipher.DecryptStruct(vaultKey, item.EncPayload, &payload); err != nil {
		return DecryptedPayload{}, fmt.Errorf("decrypt payload: %w", err)
	}
	return DecryptedPayload{ID: item.ID, Version: item.Version, Payload: payload}, nil
}
