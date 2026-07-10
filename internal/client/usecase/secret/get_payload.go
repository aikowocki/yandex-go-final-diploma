package secret

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// GetPayload возвращает расшифрованный Tier 3-payload секрета. Сначала пробует локальный кеш
// (payload_loaded=1); если payload ещё не подгружен — дёргает сервер (GetPayload), кеширует
// enc_payload в localstore и дальше отдаёт из кеша. vaultID нужен для выбора VaultKey.
func (u *UseCase) GetPayload(ctx context.Context, vaultID, secretID string) (DecryptedPayload, error) {
	if secretID == "" {
		return DecryptedPayload{}, ErrEmptySecretID
	}

	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return DecryptedPayload{}, err
	}

	encPayload, version, err := u.payloadCiphertext(ctx, token, secretID)
	if err != nil {
		return DecryptedPayload{}, err
	}

	var payload secretcontent.LoginPasswordPayload
	if err := u.cipher.DecryptStruct(vaultKey, encPayload, &payload); err != nil {
		return DecryptedPayload{}, fmt.Errorf("decrypt payload: %w", err)
	}
	return DecryptedPayload{ID: secretID, Version: version, Payload: payload}, nil
}

// payloadCiphertext отдаёт enc_payload либо из локального кеша, либо тянет с сервера и кеширует.
func (u *UseCase) payloadCiphertext(ctx context.Context, token, secretID string) ([]byte, int64, error) {
	if sec, ok, err := u.local.GetSecret(ctx, secretID); err != nil {
		return nil, 0, err
	} else if ok && sec.PayloadLoaded {
		return sec.EncPayload, sec.Version, nil
	}

	item, err := u.server.GetSecretPayload(ctx, token, secretID)
	if err != nil {
		return nil, 0, err
	}

	// Кешируем payload для будущих обращений (best-effort: строка секрета может быть ещё не в кеше).
	if _, ok, gerr := u.local.GetSecret(ctx, secretID); gerr == nil && ok {
		if err := u.local.SetSecretPayload(ctx, secretID, item.EncPayload, item.Version); err != nil {
			return nil, 0, err
		}
	}
	return item.EncPayload, item.Version, nil
}
