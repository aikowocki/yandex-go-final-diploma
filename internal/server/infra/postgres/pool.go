package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB оборачивает пул соединений pgx.
type DB struct {
	*pgxpool.Pool
}

// NewPool создаёт пул соединений с БД по DSN и проверяет доступность через Ping.
func NewPool(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)

	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{pool}, nil
}
