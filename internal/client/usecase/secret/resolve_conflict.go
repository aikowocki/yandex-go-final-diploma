package secret

import (
	"context"
	"errors"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// DeleteSecret выполняет soft-delete с оптимистичной блокировкой. При успехе убирает секрет из
// локального кеша; при конфликте возвращает *GenericConflict (IsDelete=true).
// Оффлайн — ставит delete в outbox (оптимистичный успех, err == nil, conflict == nil).
func (u *UseCase) DeleteSecret(ctx context.Context, vaultID, secretID string, baseVersion int64) (*GenericConflict, error) {
	if secretID == "" {
		return nil, ErrEmptySecretID
	}
	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return nil, err
	}

	err = u.server.DeleteSecret(ctx, token, secretID, baseVersion)
	if err != nil {
		var conflict *grpcclient.ConflictError
		switch {
		case errors.As(err, &conflict):
			return buildDeleteConflict(u, vaultKey, vaultID, secretID, conflict.Server)
		case shouldFallbackOffline(err):
			return nil, u.deleteOffline(ctx, secretID, vaultID, baseVersion)
		default:
			return nil, err
		}
	}

	if err := u.local.DeleteSecret(ctx, secretID); err != nil {
		return nil, err
	}
	return nil, nil
}
