package localstore

import (
	"context"
	"fmt"
)

// WipeAccountData удаляет все закешированные данные (папки/секреты/оффлайн-очередь/kv-настройки).
func (s *Store) WipeAccountData(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("localstore: wipe: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // ошибка после Commit не важна

	for _, table := range []string{"outbox", "secrets", "vaults", "kv"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("localstore: wipe %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("localstore: wipe: commit: %w", err)
	}
	return nil
}
