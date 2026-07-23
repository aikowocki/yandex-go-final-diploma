package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
)

type retryingDB struct {
	pool *pgxpool.Pool
}

var _ gen.DBTX = retryingDB{}

func (d retryingDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	var tag pgconn.CommandTag
	err := withRetry(ctx, isConnRetryable, func() error {
		var e error
		tag, e = d.pool.Exec(ctx, sql, args...)
		return e
	})
	return tag, err
}

func (d retryingDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	var rows pgx.Rows
	err := withRetry(ctx, isConnRetryable, func() error {
		var e error
		rows, e = d.pool.Query(ctx, sql, args...)
		return e
	})
	return rows, err
}

func (d retryingDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	// QueryRow не возвращает ошибку немедленно (она всплывёт на Scan), поэтому
	// ретраить по коннекту нужно на самом получении строки через Query.
	rows, err := d.Query(ctx, sql, args...)
	return queryRow{rows: rows, err: err}
}

// queryRow адаптирует результат Query к интерфейсу pgx.Row, пробрасывая
// ошибку соединения (если ретраи не помогли) в Scan.
type queryRow struct {
	rows pgx.Rows
	err  error
}

func (r queryRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	defer r.rows.Close()
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return err
		}
		return pgx.ErrNoRows
	}
	if err := r.rows.Scan(dest...); err != nil {
		return err
	}
	return r.rows.Err()
}
