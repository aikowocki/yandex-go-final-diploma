package localstore

import (
	"context"
	"database/sql"
	"errors"
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
	status := e.Status
	if status == "" {
		status = contracts.OutboxStatusPending
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO outbox (op, entity, entity_id, base_version, payload, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(e.Op), e.Entity, e.EntityID, e.BaseVersion, e.Payload, string(status), createdAt)
	if err != nil {
		return 0, fmt.Errorf("localstore: enqueue outbox: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("localstore: enqueue outbox id: %w", err)
	}
	return id, nil
}

// ListPendingOutbox возвращает записи очереди со статусом pending в порядке добавления (FIFO).
func (s *Store) ListPendingOutbox(ctx context.Context) ([]contracts.OutboxEntry, error) {
	return s.listOutbox(ctx, `WHERE status = ?`, string(contracts.OutboxStatusPending))
}

// ListOutboxByStatus возвращает записи очереди с указанным статусом (FIFO).
func (s *Store) ListOutboxByStatus(ctx context.Context, status contracts.OutboxStatus) ([]contracts.OutboxEntry, error) {
	return s.listOutbox(ctx, `WHERE status = ?`, string(status))
}

func (s *Store) listOutbox(ctx context.Context, where string, args ...any) ([]contracts.OutboxEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, op, entity, entity_id, base_version, payload, status, created_at
		FROM outbox `+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, fmt.Errorf("localstore: list outbox: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []contracts.OutboxEntry
	for rows.Next() {
		e, err := scanOutbox(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// GetOutbox возвращает одну запись очереди по id.
func (s *Store) GetOutbox(ctx context.Context, id int64) (contracts.OutboxEntry, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, op, entity, entity_id, base_version, payload, status, created_at
		FROM outbox WHERE id = ?`, id)
	e, err := scanOutbox(row)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.OutboxEntry{}, false, nil
	}
	if err != nil {
		return contracts.OutboxEntry{}, false, fmt.Errorf("localstore: get outbox: %w", err)
	}
	return e, true, nil
}

// SetOutboxStatus меняет статус записи очереди.
func (s *Store) SetOutboxStatus(ctx context.Context, id int64, status contracts.OutboxStatus) error {
	res, err := s.db.ExecContext(ctx, `UPDATE outbox SET status = ? WHERE id = ?`, string(status), id)
	if err != nil {
		return fmt.Errorf("localstore: set outbox status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("localstore: set outbox status: entry %d not found", id)
	}
	return nil
}

// RemoveOutbox удаляет запись очереди по id.
func (s *Store) RemoveOutbox(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM outbox WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("localstore: remove outbox: %w", err)
	}
	return nil
}

func scanOutbox(sc scanner) (contracts.OutboxEntry, error) {
	var (
		e      contracts.OutboxEntry
		op     string
		status string
	)
	if err := sc.Scan(&e.ID, &op, &e.Entity, &e.EntityID, &e.BaseVersion, &e.Payload, &status, &e.CreatedAt); err != nil {
		return contracts.OutboxEntry{}, err
	}
	e.Op = contracts.OutboxOp(op)
	e.Status = contracts.OutboxStatus(status)
	return e, nil
}
