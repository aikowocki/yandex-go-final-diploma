package secret

import (
	"context"

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
// затем обогащает полями row (Tier 2a) и index (Tier 2b). Все три тира расшифровываются
// VaultKey'ом открытой папки. Именно здесь payload тянется с сервера (лениво, при открытии карточки).
func (u *UseCase) GetDetail(ctx context.Context, vaultID, secretID string) (Detail, error) {
	// GetPayload — авторитетный источник существования/владения (сервер отдаёт ErrSecretNotFound).
	payload, err := u.GetPayload(ctx, vaultID, secretID)
	if err != nil {
		return Detail{}, err
	}

	detail := Detail{ID: payload.ID, Version: payload.Version, Payload: payload.Payload}

	rows, err := u.ListRow(ctx, vaultID)
	if err != nil {
		return Detail{}, err
	}
	for _, r := range rows {
		if r.ID == secretID {
			detail.Row = r.Row
			break
		}
	}

	indexes, err := u.ListIndex(ctx, vaultID)
	if err != nil {
		return Detail{}, err
	}
	for _, ix := range indexes {
		if ix.ID == secretID {
			detail.Index = ix.Index
			break
		}
	}

	return detail, nil
}
