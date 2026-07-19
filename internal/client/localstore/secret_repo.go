package localstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// UpsertSecretRow вставляет/обновляет Tier 2a (enc_row/type/version + флаги). При конфликте
// закешированные enc_index/enc_payload и их loaded-флаги сбрасываются, ЕСЛИ версия изменилась —
// иначе кеш содержал бы payload, зашифрованный под старую версию (AAD включает version), а
// payloadCiphertext вернул бы его с новой version → decrypt failure.
func (s *Store) UpsertSecretRow(ctx context.Context, sec contracts.LocalSecret) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO secrets (id, vault_id, type, enc_row, enc_index, enc_payload,
			version, index_loaded, payload_loaded, dirty, deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			vault_id = excluded.vault_id,
			type     = excluded.type,
			enc_row  = excluded.enc_row,
			version  = excluded.version,
			dirty    = excluded.dirty,
			deleted  = excluded.deleted,
			payload_loaded = CASE WHEN secrets.version != excluded.version THEN 0 ELSE secrets.payload_loaded END,
			index_loaded   = CASE WHEN secrets.version != excluded.version THEN 0 ELSE secrets.index_loaded END,
			enc_payload    = CASE WHEN secrets.version != excluded.version THEN NULL ELSE secrets.enc_payload END,
			enc_index      = CASE WHEN secrets.version != excluded.version THEN NULL ELSE secrets.enc_index END`,
		sec.ID, sec.VaultID, sec.Type, sec.EncRow, sec.EncIndex, sec.EncPayload,
		sec.Version, boolToInt(sec.IndexLoaded), boolToInt(sec.PayloadLoaded),
		boolToInt(sec.Dirty), boolToInt(sec.Deleted))
	if err != nil {
		return fmt.Errorf("localstore: upsert secret row: %w", err)
	}
	return nil
}

// SetSecretPayload кеширует Tier 3 (enc_payload) и выставляет payload_loaded=1.
// Обновляет только если version в БД совпадает с переданной (enc_payload привязан к конкретной
// version через AAD) — иначе игнорирует (stale payload, следующий sync подтянет).
func (s *Store) SetSecretPayload(ctx context.Context, id string, encPayload []byte, version int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE secrets SET enc_payload = ?, payload_loaded = 1 WHERE id = ? AND version = ?`,
		encPayload, id, version)
	if err != nil {
		return fmt.Errorf("localstore: set secret payload: %w", err)
	}
	_ = res
	return nil
}

// SetSecretIndex кеширует Tier 2b (enc_index) и выставляет index_loaded=1.
// Обновляет только если version в БД совпадает с переданной (enc_index привязан к конкретной
// version через AAD) — иначе игнорирует (stale index от прошлого RPC, следующий sync подтянет).
func (s *Store) SetSecretIndex(ctx context.Context, id string, encIndex []byte, version int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE secrets SET enc_index = ?, index_loaded = 1 WHERE id = ? AND version = ?`,
		encIndex, id, version)
	if err != nil {
		return fmt.Errorf("localstore: set secret index: %w", err)
	}
	// Если version не совпала (n==0) — не ошибка: секрет мог обновиться между pullVaultRows
	// и LoadIndexes, следующий sync-цикл подтянет свежий индекс.
	_ = res
	return nil
}

// ListSecretsByVault возвращает все неудалённые секреты указанного vault.
func (s *Store) ListSecretsByVault(ctx context.Context, vaultID string) ([]contracts.LocalSecret, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vault_id, type, enc_row, enc_index, enc_payload,
			version, index_loaded, payload_loaded, dirty, deleted
		FROM secrets WHERE vault_id = ? AND deleted = 0 ORDER BY id`, vaultID)
	if err != nil {
		return nil, fmt.Errorf("localstore: list secrets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []contracts.LocalSecret
	for rows.Next() {
		sec, err := scanSecret(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, sec)
	}
	return result, rows.Err()
}

// GetSecret возвращает секрет по id, если он существует.
func (s *Store) GetSecret(ctx context.Context, id string) (contracts.LocalSecret, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, vault_id, type, enc_row, enc_index, enc_payload,
			version, index_loaded, payload_loaded, dirty, deleted
		FROM secrets WHERE id = ?`, id)
	sec, err := scanSecret(row)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.LocalSecret{}, false, nil
	}
	if err != nil {
		return contracts.LocalSecret{}, false, fmt.Errorf("localstore: get secret: %w", err)
	}
	return sec, true, nil
}

// DeleteSecret удаляет секрет по id.
func (s *Store) DeleteSecret(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM secrets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("localstore: delete secret: %w", err)
	}
	return nil
}

func scanSecret(sc scanner) (contracts.LocalSecret, error) {
	var sec contracts.LocalSecret
	var indexLoaded, payloadLoaded, dirty, deleted int
	if err := sc.Scan(
		&sec.ID, &sec.VaultID, &sec.Type, &sec.EncRow, &sec.EncIndex, &sec.EncPayload,
		&sec.Version, &indexLoaded, &payloadLoaded, &dirty, &deleted,
	); err != nil {
		return contracts.LocalSecret{}, err
	}
	sec.IndexLoaded = indexLoaded != 0
	sec.PayloadLoaded = payloadLoaded != 0
	sec.Dirty = dirty != 0
	sec.Deleted = deleted != 0
	return sec, nil
}
