package app

import (
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app/providers"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/keyring"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/logger"
)

// New собирает все зависимости из уже разобранного конфига и возвращает готовый Container.
// Конфиг парсится в точке входа (kong.Parse корневой CLI-структуры) и передаётся сюда.
func New(cfg *config.ClientConfig) (*Container, error) {
	logger.Setup(cfg.LogLevel)

	grpcClient, err := grpcclient.New(cfg.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("grpc client: %w", err)
	}

	tokenStore := keyring.New(cfg.DataDir, !cfg.NoPersist)
	crypto := cryptoimpl.Crypto{}
	authUseCase := authuc.New(grpcClient, crypto, tokenStore)

	localizer := providers.NewLocalizer(cfg)

	return &Container{
		Config:    cfg,
		GRPC:      grpcClient,
		Auth:      authUseCase,
		Localizer: localizer,
	}, nil
}
