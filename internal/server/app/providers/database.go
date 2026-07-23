package providers

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
)

// NewDatabase создаёт пул подключений к Postgres и выполняет миграции.
func NewDatabase(ctx context.Context, cfg *config.ServerConfig) (*postgres.DB, error) {
	db, err := postgres.NewPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	if err := postgres.RunMigrations(cfg.DatabaseDSN); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
