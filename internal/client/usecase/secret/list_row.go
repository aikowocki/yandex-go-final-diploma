package secret

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// ListRow возвращает расшифрованные Tier 2a-строки секретов папки из ЛОКАЛЬНОГО кеша
// (без сетевых вызовов). Наполнение кеша — задача sync engine. enc_payload здесь не трогается.
func (u *UseCase) ListRow(ctx context.Context, vaultID string) ([]DecryptedRow, error) {
	if vaultID == "" {
		return nil, ErrEmptyVaultID
	}
	vaultKey, err := u.vaultKey(vaultID)
	if err != nil {
		return nil, err
	}

	items, err := u.local.ListSecretsByVault(ctx, vaultID)
	if err != nil {
		return nil, err
	}

	result := make([]DecryptedRow, 0, len(items))
	for _, it := range items {
		var row secretcontent.LoginPasswordRow
		ad := secretAAD(vaultID, it.ID, it.Version, tierRow)
		if err := u.cipher.DecryptStruct(vaultKey, ad, it.EncRow, &row); err != nil {
			return nil, fmt.Errorf("decrypt row: %w", err)
		}
		result = append(result, DecryptedRow{ID: it.ID, Version: it.Version, Row: row})
	}
	return result, nil
}
