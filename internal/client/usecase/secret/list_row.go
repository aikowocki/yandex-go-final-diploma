package secret

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// ListRow возвращает расшифрованные Tier 2a-строки секретов ваулта. enc_payload не запрашивается
// и не расшифровывается (пароль показывается только по GetPayload).
func (u *UseCase) ListRow(ctx context.Context, vaultID string) ([]DecryptedRow, error) {
	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return nil, err
	}

	items, err := u.server.ListSecretRows(ctx, token, vaultID)
	if err != nil {
		return nil, err
	}

	result := make([]DecryptedRow, 0, len(items))
	for _, it := range items {
		var row secretcontent.LoginPasswordRow
		if err := u.cipher.DecryptStruct(vaultKey, it.EncRow, &row); err != nil {
			return nil, fmt.Errorf("decrypt row: %w", err)
		}
		result = append(result, DecryptedRow{ID: it.ID, Version: it.Version, Row: row})
	}
	return result, nil
}
