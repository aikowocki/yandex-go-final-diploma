package secret

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// LoadIndexes догружает Tier 2b (enc_index) для папки с сервера и кеширует его в localstore
// (index_loaded=1). Предназначена для фонового вызова (отдельная горутина) после того как
// Tier 2a (строки списка) уже отображён: расширяет поиск на note/custom_fields. Идемпотентна.
// Секреты с локальными несинхронизированными изменениями (dirty) пропускаются, чтобы не
// затереть их более старым серверным индексом.
func (u *UseCase) LoadIndexes(ctx context.Context, vaultID string) error {
	if vaultID == "" {
		return ErrEmptyVaultID
	}
	token, err := u.accessToken()
	if err != nil {
		return err
	}

	items, err := u.server.ListSecretIndex(ctx, token, vaultID)
	if err != nil {
		return err
	}

	for _, it := range items {
		local, ok, err := u.local.GetSecret(ctx, it.ID)
		if err != nil {
			return err
		}
		// Кешируем индекс только для уже известных (через Tier 2a) и не-dirty секретов.
		if !ok || local.Dirty || local.IndexLoaded {
			continue
		}
		if err := u.local.SetSecretIndex(ctx, it.ID, it.EncIndex, it.Version); err != nil {
			return err
		}
	}
	return nil
}

// decryptIndex расшифровывает Tier 2b-индекс из локального кеша для одного секрета.
func (u *UseCase) decryptIndex(vaultKey []byte, vaultID string, sec localSecretView) (secretcontent.LoginPasswordIndex, error) {
	var idx secretcontent.LoginPasswordIndex
	if len(sec.EncIndex) == 0 {
		return idx, nil
	}
	ad := secretAAD(vaultID, sec.ID, sec.Version, tierIndex)
	if err := u.cipher.DecryptStruct(vaultKey, ad, sec.EncIndex, &idx); err != nil {
		return idx, fmt.Errorf("decrypt index: %w", err)
	}
	return idx, nil
}

// localSecretView — минимальная проекция локального секрета для расшифровки индекса.
type localSecretView struct {
	ID       string
	Version  int64
	EncIndex []byte
}
