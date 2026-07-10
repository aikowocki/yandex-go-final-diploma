package secret

import (
	"context"
	"errors"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// ResolveConflict применяет выбор пользователя к конфликту версий:
//   - ChoiceMine   — оставить мою версию: повторяет операцию (update/delete), отталкиваясь от
//     server.Version (перезатирает серверную). Может снова вернуть *ConflictResult при новой гонке.
//   - ChoiceServer — принять серверную версию: обновляет локальный кеш серверными данными,
//     локальные изменения отбрасываются (dirty снимается).
func (u *UseCase) ResolveConflict(ctx context.Context, conflict *ConflictResult, choice ConflictChoice) (*ConflictResult, error) {
	if conflict == nil {
		return nil, ErrNilConflict
	}

	switch choice {
	case ChoiceMine:
		if conflict.isDelete {
			return u.DeleteSecret(ctx, conflict.vaultID, conflict.SecretID, conflict.server.Version)
		}
		return u.UpdateLoginPassword(ctx, conflict.vaultID, conflict.SecretID, conflict.server.Version, conflict.mineInput)
	case ChoiceServer:
		return nil, u.acceptServerVersion(ctx, conflict)
	default:
		return nil, ErrUnknownChoice
	}
}

// acceptServerVersion записывает серверную версию секрета в локальный кеш, отбрасывая
// локальные изменения (dirty=false), используя серверные блобы, полученные при конфликте.
func (u *UseCase) acceptServerVersion(ctx context.Context, conflict *ConflictResult) error {
	secretType := conflict.server.Type
	if secretType == 0 {
		secretType = int32(domain.SecretTypeLoginPassword)
	}
	return u.cacheFullSecret(ctx, conflict.SecretID, conflict.vaultID, secretType,
		conflict.server.EncRow, conflict.server.EncIndex, conflict.server.EncPayload,
		conflict.server.Version, false)
}

// DeleteSecret выполняет soft-delete с оптимистичной блокировкой. При успехе убирает секрет из
// локального кеша; при конфликте возвращает *ConflictResult (isDelete=true). Оффлайн — ставит
// delete в outbox (оптимистичный успех, err == nil, ConflictResult == nil).
func (u *UseCase) DeleteSecret(ctx context.Context, vaultID, secretID string, baseVersion int64) (*ConflictResult, error) {
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
			res, berr := u.buildConflict(vaultKey, vaultID, secretID, baseVersion, CreateLoginPasswordInput{}, conflict.Server)
			if berr != nil {
				return nil, berr
			}
			res.isDelete = true
			return res, nil
		case errors.Is(err, grpcclient.ErrUnavailable):
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
