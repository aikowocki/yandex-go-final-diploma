package localstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// UpsertVault вставляет/обновляет метаданные папки. synced_version существующей строки
// сохраняется (ON CONFLICT не трогает его — им управляет только sync engine).
func (s *Store) UpsertVault(ctx context.Context, v contracts.LocalVault) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vaults (id, wrapped_vault_key, enc_name, version, synced_version, deleted)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			wrapped_vault_key = excluded.wrapped_vault_key,
			enc_name          = excluded.enc_name,
			version           = excluded.version,
			deleted           = excluded.deleted`,
		v.ID, v.WrappedVaultKey, v.EncName, v.Version, v.SyncedVersion, boolToInt(v.Deleted))
	if err != nil {
		return fmt.Errorf("localstore: upsert vault: %w", err)
	}
	return nil
}

func (s *Store) ListVaults(ctx context.Context) ([]contracts.LocalVault, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, wrapped_vault_key, enc_name, version, synced_version, deleted
		FROM vaults WHERE deleted = 0 ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("localstore: list vaults: %w", err)
	}
	defer rows.Close()

	var result []contracts.LocalVault
	for rows.Next() {
		v, err := scanVault(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (s *Store) GetVault(ctx context.Context, id string) (contracts.LocalVault, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, wrapped_vault_key, enc_name, version, synced_version, deleted
		FROM vaults WHERE id = ?`, id)
	v, err := scanVault(row)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.LocalVault{}, false, nil
	}
	if err != nil {
		return contracts.LocalVault{}, false, fmt.Errorf("localstore: get vault: %w", err)
	}
	return v, true, nil
}

func (s *Store) SetVaultSyncedVersion(ctx context.Context, id string, syncedVersion int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE vaults SET synced_version = ? WHERE id = ?`, syncedVersion, id)
	if err != nil {
		return fmt.Errorf("localstore: set synced_version: %w", err)
	}
	return nil
}

func (s *Store) DeleteVault(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM vaults WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("localstore: delete vault: %w", err)
	}
	return nil
}

func scanVault(sc scanner) (contracts.LocalVault, error) {
	var v contracts.LocalVault
	var deleted int
	if err := sc.Scan(&v.ID, &v.WrappedVaultKey, &v.EncName, &v.Version, &v.SyncedVersion, &deleted); err != nil {
		return contracts.LocalVault{}, err
	}
	v.Deleted = deleted != 0
	return v, nil
}
