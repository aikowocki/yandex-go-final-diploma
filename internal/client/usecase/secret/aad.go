package secret

import "fmt"

// Тиры секрета — метка в AAD, чтобы блоб одного тира нельзя было подставить в слот другого.
const (
	tierRow     = "row"
	tierIndex   = "index"
	tierPayload = "payload"
)

// secretAAD формирует детерминированный associated data для AEAD-шифрования полей секрета.
// Контекст привязывает шифротекст к папке, конкретному секрету, его версии и тиру:
//   - vault_id/secret_id — защита от подмены блобов между секретами (в т.ч. внутри одной папки);
//   - version — защита от отката на старую версию (anti-rollback);
//   - tier — защита от подстановки enc_row вместо enc_payload и т.п.
//
// AAD должна детерминированно воспроизводиться на записи и чтении и НЕ храниться внутри блоба.
func secretAAD(vaultID, secretID string, version int64, tier string) []byte {
	return []byte(fmt.Sprintf("gophkeeper:secret:v1|vault=%s|secret=%s|ver=%d|tier=%s",
		vaultID, secretID, version, tier))
}
