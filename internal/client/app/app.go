package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app/providers"
	"github.com/aikowocki/yandex-go-final-diploma/internal/logger"
)

// New собирает все зависимости и возвращает готовый Container.
func New() (*Container, error) {
	cfg, err := providers.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	logger.Setup(cfg.LogLevel)

	grpcClient, err := providers.NewGRPCClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("grpc client: %w", err)
	}

	localizer := providers.NewLocalizer(cfg)

	return &Container{
		Config:    cfg,
		GRPC:      grpcClient,
		Localizer: localizer,
	}, nil
}

// Run выполняет тестовый ping-запрос к серверу и локализованный тестовый вывод.
func Run() error {
	c, err := New()
	if err != nil {
		return err
	}
	defer c.Close()

	println(c.Localizer.T("test"))

	msg, err := c.GRPC.Ping(context.Background())
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	slog.Info("ping", "response", msg)
	return nil
}
