package localstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// KVGet возвращает значение настройки по ключу. Второй результат — признак наличия ключа.
func (s *Store) KVGet(ctx context.Context, key string) ([]byte, bool, error) {
	var v []byte
	err := s.db.QueryRowContext(ctx, `SELECT v FROM kv WHERE k = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("localstore: kv get: %w", err)
	}
	return v, true, nil
}

// KVSet вставляет/обновляет значение настройки.
func (s *Store) KVSet(ctx context.Context, key string, value []byte) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO kv (k, v) VALUES (?, ?)
		ON CONFLICT(k) DO UPDATE SET v = excluded.v`, key, value)
	if err != nil {
		return fmt.Errorf("localstore: kv set: %w", err)
	}
	return nil
}
