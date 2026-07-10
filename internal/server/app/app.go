package app

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/logger"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/app/providers"
)

// New собирает все зависимости и возвращает готовый Container.
func New(ctx context.Context) (*Container, error) {
	cfg, err := providers.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	logger.Setup(cfg.LogLevel)

	db, err := providers.NewDatabase(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	objectStore, err := providers.NewObjectStore(ctx, cfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("object store: %w", err)
	}

	grpcSrv := providers.NewGRPC(cfg, db, objectStore)

	return &Container{
		Config:      cfg,
		DB:          db,
		ObjectStore: objectStore,
		GRPC:        grpcSrv,
	}, nil
}
