package localstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// UpsertSecretRow вставляет/обновляет Tier 2a (enc_row/type/version + флаги). При конфликте
// закешированные enc_index/enc_payload и их loaded-флаги НЕ затираются (обновляем только строку).
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
			deleted  = excluded.deleted`,
		sec.ID, sec.VaultID, sec.Type, sec.EncRow, sec.EncIndex, sec.EncPayload,
		sec.Version, boolToInt(sec.IndexLoaded), boolToInt(sec.PayloadLoaded),
		boolToInt(sec.Dirty), boolToInt(sec.Deleted))
	if err != nil {
		return fmt.Errorf("localstore: upsert secret row: %w", err)
	}
	return nil
}

// SetSecretPayload кеширует Tier 3 (enc_payload) и выставляет payload_loaded=1.
func (s *Store) SetSecretPayload(ctx context.Context, id string, encPayload []byte, version int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE secrets SET enc_payload = ?, payload_loaded = 1, version = ? WHERE id = ?`,
		encPayload, version, id)
	if err != nil {
		return fmt.Errorf("localstore: set secret payload: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("localstore: set secret payload: secret %q not found", id)
	}
	return nil
}

func (s *Store) ListSecretsByVault(ctx context.Context, vaultID string) ([]contracts.LocalSecret, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vault_id, type, enc_row, enc_index, enc_payload,
			version, index_loaded, payload_loaded, dirty, deleted
		FROM secrets WHERE vault_id = ? AND deleted = 0 ORDER BY id`, vaultID)
	if err != nil {
		return nil, fmt.Errorf("localstore: list secrets: %w", err)
	}
	defer rows.Close()

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
