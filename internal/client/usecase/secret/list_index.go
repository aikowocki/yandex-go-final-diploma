package secret

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// DecryptedIndex — расшифрованный Tier 2b-индекс секрета (для фонового поиска).
type DecryptedIndex struct {
	ID      string
	Version int64
	Index   secretcontent.LoginPasswordIndex
}

// ListIndex возвращает расшифрованные Tier 2b-индексы секретов ваулта.
func (u *UseCase) ListIndex(ctx context.Context, vaultID string) ([]DecryptedIndex, error) {
	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return nil, err
	}

	items, err := u.server.ListSecretIndex(ctx, token, vaultID)
	if err != nil {
		return nil, err
	}

	result := make([]DecryptedIndex, 0, len(items))
	for _, it := range items {
		var idx secretcontent.LoginPasswordIndex
		if err := u.cipher.DecryptStruct(vaultKey, it.EncIndex, &idx); err != nil {
			return nil, fmt.Errorf("decrypt index: %w", err)
		}
		result = append(result, DecryptedIndex{ID: it.ID, Version: it.Version, Index: idx})
	}
	return result, nil
}
