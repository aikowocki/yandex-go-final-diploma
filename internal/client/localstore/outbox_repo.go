package localstore

import (
	"context"
	"fmt"
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// EnqueueOutbox добавляет отложенную операцию в очередь, возвращает её id.
func (s *Store) EnqueueOutbox(ctx context.Context, e contracts.OutboxEntry) (int64, error) {
	createdAt := e.CreatedAt
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO outbox (op, entity, entity_id, base_version, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		string(e.Op), e.Entity, e.EntityID, e.BaseVersion, e.Payload, createdAt)
	if err != nil {
		return 0, fmt.Errorf("localstore: enqueue outbox: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("localstore: enqueue outbox id: %w", err)
	}
	return id, nil
}

// ListPendingOutbox возвращает записи очереди в порядке добавления (FIFO).
func (s *Store) ListPendingOutbox(ctx context.Context) ([]contracts.OutboxEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, op, entity, entity_id, base_version, payload, created_at
		FROM outbox ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("localstore: list outbox: %w", err)
	}
	defer rows.Close()

	var result []contracts.OutboxEntry
	for rows.Next() {
		var (
			e  contracts.OutboxEntry
			op string
		)
		if err := rows.Scan(&e.ID, &op, &e.Entity, &e.EntityID, &e.BaseVersion, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("localstore: scan outbox: %w", err)
		}
		e.Op = contracts.OutboxOp(op)
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *Store) RemoveOutbox(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM outbox WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("localstore: remove outbox: %w", err)
	}
	return nil
}
