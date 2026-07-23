package secret

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// Detail — полная расшифрованная карточка секрета (все тиры вместе).
type Detail struct {
	ID      string
	Version int64
	Row     secretcontent.LoginPasswordRow
	Index   secretcontent.LoginPasswordIndex
	Payload secretcontent.LoginPasswordPayload
}

// GetDetail собирает полную карточку: payload (Tier 3, авторитетная проверка существования),
// затем row (Tier 2a) из локального кеша и index (Tier 2b) — при необходимости догружает его.
// Все тиры расшифровываются VaultKey'ом с AAD-контекстом (vault_id|secret_id|version|tier).
func (u *UseCase) GetDetail(ctx context.Context, vaultID, secretID string) (Detail, error) {
	vaultKey, err := u.vaultKey(vaultID)
	if err != nil {
		return Detail{}, err
	}

	// GetPayload — авторитетный источник существования/владения (сервер отдаёт ErrSecretNotFound).
	payload, err := u.GetPayload(ctx, vaultID, secretID)
	if err != nil {
		return Detail{}, err
	}
	detail := Detail{ID: payload.ID, Version: payload.Version, Payload: payload.Payload}

	local, ok, err := u.local.GetSecret(ctx, secretID)
	if err != nil {
		return Detail{}, err
	}
	if !ok {
		return detail, nil
	}

	ad := secretAAD(vaultID, secretID, local.Version, tierRow)
	if err := u.cipher.DecryptStruct(vaultKey, ad, local.EncRow, &detail.Row); err != nil {
		return Detail{}, fmt.Errorf("decrypt row: %w", err)
	}

	// Догружаем Tier 2b, если ещё не загружен (best-effort — при ошибке просто без индекса).
	if !local.IndexLoaded {
		if err := u.LoadIndexes(ctx, vaultID); err == nil {
			local, ok, _ = u.local.GetSecret(ctx, secretID)
		}
	}
	if ok && local.IndexLoaded {
		idx, err := u.decryptIndex(vaultKey, vaultID, localSecretView{ID: local.ID, Version: local.Version, EncIndex: local.EncIndex})
		if err != nil {
			return Detail{}, err
		}
		detail.Index = idx
	}
	return detail, nil
}
